// Package cmd wires the mask CLI together with cobra.
//
// Commands that change the shell environment (`switch`, `off`) print shell
// code to stdout; the wrapper function installed by `mask init` pipes that
// stdout into `source`/`eval`. All other output and all errors go to stderr so
// they can never be mistaken for shell code.
package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/metruzanca/mask/internal/config"
	"github.com/metruzanca/mask/internal/hookops"
	"github.com/metruzanca/mask/internal/maskops"
	"github.com/metruzanca/mask/internal/repo"
	"github.com/metruzanca/mask/internal/shellout"
	"github.com/metruzanca/mask/internal/state"
	"github.com/metruzanca/mask/internal/ui"
)

// Execute runs the root command and returns a process exit code. Errors are
// printed to stderr here, since the root command silences cobra's own error
// output to keep stderr free of duplicate messages.
func Execute() int {
	root := newRoot()
	if err := root.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "mask: %v\n", err)
		return 1
	}
	return 0
}

func newRoot() *cobra.Command {
	root := &cobra.Command{
		Use:   "mask",
		Short: "Wear a git identity and SSH key for the current shell",
		Long: "Mask swaps your git user.name/email and the SSH key git push uses,\n" +
			"so you can commit and push as different people without editing git\n" +
			"config. Install the shell wrapper once with `mask init <shell>`.",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatus(cmd)
		},
	}

	root.PersistentFlags().String("shell", "", "shell to emit code for (set by the installed wrapper)")
	root.AddCommand(newCreate())
	root.AddCommand(newSwitch())
	root.AddCommand(newOff())
	root.AddCommand(newList())
	root.AddCommand(newInit())
	root.AddCommand(newSettings())
	root.AddCommand(newVersion())
	root.AddCommand(newHook())
	return root
}

func newCreate() *cobra.Command {
	return &cobra.Command{
		Use:   "create",
		Short: "Create a new mask with a huh form",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !term.IsTerminal(int(os.Stdin.Fd())) {
				return errors.New("`mask create` needs an interactive terminal")
			}
			res, err := ui.CreateForm()
			if err != nil {
				return err
			}
			cfg, err := loadOrInit()
			if err != nil {
				return err
			}
			if _, exists := cfg.Get(res.Name); exists {
				return fmt.Errorf("mask %q already exists", res.Name)
			}
			cfg.Set(res.Name, config.Mask{
				GitName:  res.GitName,
				GitEmail: res.GitEmail,
				SSHKey:   res.SSHKey,
			})
			if err := cfg.Save(); err != nil {
				return err
			}
			path, _ := config.Path()
			fmt.Fprintf(cmd.ErrOrStderr(), "Created mask %q in %s\n", res.Name, path)
			fmt.Fprintf(cmd.ErrOrStderr(), "Add env vars by editing the [mask.%s.env] table, then run `mask switch`.\n", res.Name)
			return nil
		},
	}
}

func newSwitch() *cobra.Command {
	return &cobra.Command{
		Use:   "switch [mask]",
		Short: "Switch to a mask (toggles between two, dropdown for three or more)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				if errors.Is(err, config.ErrNotExist) {
					return errors.New("no masks defined; run `mask create` first")
				}
				return err
			}
			requested := ""
			if len(args) == 1 {
				requested = args[0]
			}
			sel, err := maskops.Select(cfg, os.Getenv("MASK_NAME"), requested)
			if err != nil {
				return err
			}
			name := sel.Name
			if sel.NeedUI {
				if !term.IsTerminal(int(os.Stdin.Fd())) {
					return fmt.Errorf("need an interactive terminal to choose among %d masks; pass a name", len(cfg.Names()))
				}
				name, err = ui.SwitchForm(cfg.Names())
				if err != nil {
					return err
				}
			}
			m, _ := cfg.Get(name)
			shell, err := resolveShell(cmd)
			if err != nil {
				return err
			}
			payload := maskops.BuildPayload(name, m)
			info, inRepo := currentRepo()
			if inRepo {
				payload.Repo = info.ID
				if err := rememberRepo(info, name); err != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "mask: could not remember repo: %v\n", err)
				}
			}
			out, err := shellout.Render(shell, payload)
			if err != nil {
				return err
			}
			_, err = io.WriteString(cmd.OutOrStdout(), out)
			return err
		},
	}
}

func newOff() *cobra.Command {
	return &cobra.Command{
		Use:   "off",
		Short: "Take the current mask off",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			shell, err := resolveShell(cmd)
			if err != nil {
				return err
			}
			if info, ok := currentRepo(); ok {
				if err := forgetRepo(info); err != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "mask: could not forget repo: %v\n", err)
				}
			}
			out, err := shellout.Render(shell, maskops.OffPayload())
			if err != nil {
				return err
			}
			_, err = io.WriteString(cmd.OutOrStdout(), out)
			return err
		},
	}
}

func newList() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List configured masks",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				if errors.Is(err, config.ErrNotExist) {
					fmt.Fprintln(cmd.OutOrStdout(), "No masks defined.")
					return nil
				}
				return err
			}
			names := cfg.Names()
			if len(names) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No masks defined.")
				return nil
			}
			current := os.Getenv("MASK_NAME")
			for _, name := range names {
				m, _ := cfg.Get(name)
				marker := "  "
				if name == current {
					marker = "* "
				}
				fmt.Fprintf(cmd.OutOrStdout(), "%s%-16s %s <%s>\n", marker, name, m.GitName, m.GitEmail)
			}
			return nil
		},
	}
}

func newInit() *cobra.Command {
	return &cobra.Command{
		Use:   "init [shell]",
		Short: "Print the shell wrapper (e.g. `mask init fish | source`)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			shellName := ""
			if len(args) == 1 {
				shellName = args[0]
			}
			shell, ok := shellout.Parse(shellName)
			if !ok {
				detected, found := shellout.DetectShell(os.Getenv)
				if !found {
					return fmt.Errorf("couldn't tell which shell you use; pass fish, bash, or zsh")
				}
				shell = detected
			}
			out, err := shellout.Init(shell)
			if err != nil {
				return err
			}
			_, err = io.WriteString(cmd.OutOrStdout(), out)
			return err
		},
	}
}

func newSettings() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "settings",
		Short: "Show or change settings (automatic per-repo masking)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadOrInit()
			if err != nil {
				return err
			}
			if len(args) == 0 {
				if !term.IsTerminal(int(os.Stdin.Fd())) {
					fmt.Fprintf(cmd.OutOrStdout(), "automatic = %t\n", cfg.Automatic())
					return nil
				}
				choice, err := ui.SettingsForm(cfg.Automatic())
				if err != nil {
					return err
				}
				cfg.SetAutomatic(choice == "automatic")
			} else {
				switch strings.ToLower(args[0]) {
				case "on", "automatic", "auto":
					cfg.SetAutomatic(true)
				case "off", "manual":
					cfg.SetAutomatic(false)
				default:
					return fmt.Errorf("unknown setting %q; use on or off", args[0])
				}
			}
			if err := cfg.Save(); err != nil {
				return err
			}
			fmt.Fprintf(cmd.ErrOrStderr(), "automatic = %t\n", cfg.Automatic())
			return nil
		},
	}
	return cmd
}

// newHook is the hidden command the shell's directory hook calls. It prints
// shell code to stdout and any notice to stderr.
func newHook() *cobra.Command {
	return &cobra.Command{
		Use:    "_hook",
		Hidden: true,
		Short:  "Internal: apply the per-repo mask on directory change",
		Args:   cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			shell, err := resolveShell(cmd)
			if err != nil {
				return err
			}

			cfg, cfgErr := config.Load()
			if cfgErr != nil {
				if errors.Is(cfgErr, config.ErrNotExist) {
					cfg = &config.Config{Masks: map[string]config.Mask{}}
				} else {
					return cfgErr
				}
			}

			st, _ := state.Load()
			info, inRepo := currentRepo()

			in := hookops.Input{
				Automatic:   cfg.Automatic(),
				CurrentMask: os.Getenv("MASK_NAME"),
				CurrentRepo: os.Getenv("MASK_REPO"),
				InRepo:      inRepo,
				Repo:        info,
				MaskExists:  func(name string) bool { _, ok := cfg.Get(name); return ok },
			}
			if inRepo {
				if cached, ok := st.Lookup(info.ID); ok {
					in.CachedMask = cached
					in.HasCache = true
				}
			}

			d := hookops.Decide(in)

			if d.Forget && inRepo {
				st.Forget(info.ID)
			}
			if inRepo && d.Action == hookops.ActionSwitch {
				st.Record(info.ID, info.Root, d.Mask)
			}
			if err := st.Save(); err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "mask: could not save repo state: %v\n", err)
			}
			if d.Log != "" {
				if err := state.AppendLog(d.Log); err != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "mask: could not write log: %v\n", err)
				}
			}
			if d.Notice != "" {
				fmt.Fprintf(cmd.ErrOrStderr(), "mask: %s\n", d.Notice)
			}

			payload := shellout.Payload{Repo: d.RepoID, RepoOnly: true}
			switch d.Action {
			case hookops.ActionSwitch:
				m, _ := cfg.Get(d.Mask)
				payload = maskops.BuildPayload(d.Mask, m)
				payload.Repo = d.RepoID
			case hookops.ActionOff:
				// The empty payload clears the mask; only the repo marker is
				// carried forward.
				payload = shellout.Payload{Repo: d.RepoID}
			}

			out, err := shellout.Render(shell, payload)
			if err != nil {
				return err
			}
			_, err = io.WriteString(cmd.OutOrStdout(), out)
			return err
		},
	}
}

// currentRepo resolves the repository containing the working directory.
func currentRepo() (repo.Info, bool) {
	wd, err := os.Getwd()
	if err != nil {
		return repo.Info{}, false
	}
	return repo.Find(wd)
}

// rememberRepo records that info wears name, so the directory hook can restore
// it next time.
func rememberRepo(info repo.Info, name string) error {
	st, err := state.Load()
	if err != nil {
		return err
	}
	st.Record(info.ID, info.Root, name)
	return st.Save()
}

// forgetRepo drops the remembered mask for info.
func forgetRepo(info repo.Info) error {
	st, err := state.Load()
	if err != nil {
		return err
	}
	if !st.Forget(info.ID) {
		return nil
	}
	return st.Save()
}

func runStatus(cmd *cobra.Command) error {
	name := os.Getenv("GIT_AUTHOR_NAME")
	email := os.Getenv("GIT_AUTHOR_EMAIL")
	if name == "" && email == "" {
		name, email = globalGitIdentity()
	}
	report := maskops.Describe(os.Getenv("MASK_NAME"), name, email)
	if _, err := io.WriteString(cmd.OutOrStdout(), maskops.RenderReport(report)); err != nil {
		return err
	}
	cfg, err := config.Load()
	if err == nil || errors.Is(err, config.ErrNotExist) {
		if cfg == nil {
			cfg = &config.Config{}
		}
		auto := "automatic"
		if !cfg.Automatic() {
			auto = "manual"
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Mode: %s\n", auto)
	}
	if info, ok := currentRepo(); ok {
		fmt.Fprintf(cmd.OutOrStdout(), "Repo: %s\n", info.Root)
		if st, err := state.Load(); err == nil {
			if cached, ok := st.Lookup(info.ID); ok {
				fmt.Fprintf(cmd.OutOrStdout(), "Remembered here: %s\n", cached)
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), "Remembered here: (none)")
			}
		}
	}
	return nil
}

// globalGitIdentity reads the effective git identity when no mask is worn, so
// `mask` can still print the git user.name. Missing git or config yields "".
func globalGitIdentity() (string, string) {
	name, _ := gitConfig("user.name")
	email, _ := gitConfig("user.email")
	return name, email
}

func gitConfig(key string) (string, error) {
	out, err := exec.Command("git", "config", "--get", key).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func loadOrInit() (*config.Config, error) {
	cfg, err := config.Load()
	if err != nil {
		if errors.Is(err, config.ErrNotExist) {
			return &config.Config{Masks: map[string]config.Mask{}}, nil
		}
		return nil, err
	}
	return cfg, nil
}

// resolveShell prefers the --shell flag set by the installed wrapper, then
// $SHELL, then shell detection. The flag matters because $SHELL names the
// login shell, which may differ from the shell currently running.
func resolveShell(cmd *cobra.Command) (shellout.Shell, error) {
	if flag, _ := cmd.Flags().GetString("shell"); flag != "" {
		if s, ok := shellout.Parse(flag); ok {
			return s, nil
		}
		return "", fmt.Errorf("unknown shell %q", flag)
	}
	if s, ok := shellout.Parse(os.Getenv("SHELL")); ok {
		return s, nil
	}
	if s, ok := shellout.DetectShell(os.Getenv); ok {
		return s, nil
	}
	return "", fmt.Errorf("couldn't tell which shell you use; set $SHELL to fish, bash, or zsh")
}
