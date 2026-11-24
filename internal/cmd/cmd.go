// Package cmd wires the mask CLI together with cobra.
//
// Commands that change the shell environment (`switch`, `on`, `off`) print
// shell code to stdout; the wrapper function installed by `mask init` pipes
// that stdout into `source`/`eval`. All other output and all errors go to
// stderr so they can never be mistaken for shell code.
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
	root.AddCommand(newOn())
	root.AddCommand(newOff())
	root.AddCommand(newList())
	root.AddCommand(newInit())
	root.AddCommand(newSettings())
	root.AddCommand(newConfig())
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
			if res.NewKey {
				if err := ui.GenerateSSHKey(res.NewKeyPath, res.GitEmail, res.Passphrase); err != nil {
					return err
				}
				fmt.Fprintf(cmd.ErrOrStderr(), "Created SSH key %s\n", res.NewKeyPath)
				fmt.Fprintf(cmd.ErrOrStderr(), "Public key %s.pub: add it to your Git host\n", res.NewKeyPath)
				if res.Passphrase != "" {
					fmt.Fprintf(cmd.ErrOrStderr(), "Unlock it for this session with `ssh-add %s`\n", res.NewKeyPath)
				}
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
				if cfg.Automatic() {
					if err := rememberRepo(info, name); err != nil {
						fmt.Fprintf(cmd.ErrOrStderr(), "mask: could not remember repo: %v\n", err)
					}
				}
			}
			out, err := shellout.Render(shell, payload)
			if err != nil {
				return err
			}
			if _, err := io.WriteString(cmd.OutOrStdout(), out); err != nil {
				return err
			}
			if cfg.Notify() {
				fmt.Fprintf(cmd.ErrOrStderr(), "mask: %s\n", maskChangeNotice("wearing", name, info, inRepo))
			}
			return nil
		},
	}
}

func newOn() *cobra.Command {
	return &cobra.Command{
		Use:   "on",
		Short: "Put the last mask back on",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				if errors.Is(err, config.ErrNotExist) {
					return errors.New("no masks defined; run `mask create` first")
				}
				return err
			}
			info, inRepo := currentRepo()
			repoMask := ""
			if inRepo {
				if st, err := state.Load(); err == nil {
					if rec, ok := st.Lookup(info.ID); ok && !rec.None {
						repoMask = rec.Mask
					}
				}
			}
			name, err := maskops.RestoreTarget(cfg, os.Getenv(shellout.LastMaskEnv), repoMask)
			if err != nil {
				return err
			}
			m, _ := cfg.Get(name)
			shell, err := resolveShell(cmd)
			if err != nil {
				return err
			}
			payload := maskops.BuildPayload(name, m)
			if inRepo {
				payload.Repo = info.ID
				if cfg.Automatic() {
					if err := rememberRepo(info, name); err != nil {
						fmt.Fprintf(cmd.ErrOrStderr(), "mask: could not remember repo: %v\n", err)
					}
				}
			}
			out, err := shellout.Render(shell, payload)
			if err != nil {
				return err
			}
			if _, err := io.WriteString(cmd.OutOrStdout(), out); err != nil {
				return err
			}
			if cfg.Notify() {
				fmt.Fprintf(cmd.ErrOrStderr(), "mask: %s\n", maskChangeNotice("wearing", name, info, inRepo))
			}
			return nil
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
			worn := os.Getenv("MASK_NAME")
			info, inRepo := currentRepo()
			notify := true
			if cfg, err := config.Load(); err == nil {
				notify = cfg.Notify()
				if cfg.Automatic() && inRepo {
					if err := rememberNone(info); err != nil {
						fmt.Fprintf(cmd.ErrOrStderr(), "mask: could not remember repo: %v\n", err)
					}
				}
			}
			out, err := shellout.Render(shell, maskops.OffPayload())
			if err != nil {
				return err
			}
			if _, err := io.WriteString(cmd.OutOrStdout(), out); err != nil {
				return err
			}
			if worn != "" && notify {
				fmt.Fprintf(cmd.ErrOrStderr(), "mask: %s\n", maskChangeNotice("took off", worn, info, inRepo))
			}
			return nil
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

// newConfig updates settings by key, git-config style. Without a value it
// prints the current value, and without a key it prints every setting.
func newConfig() *cobra.Command {
	return &cobra.Command{
		Use:   "config [key] [value]",
		Short: "Show or change config values",
		Long: "Show or change values in the mask config file. The file is created\n" +
			"when it does not exist yet.\n\n" +
			"Available settings:\n" +
			"  setting.automatic   Apply each repo's remembered mask on directory change (true/false)\n" +
			"  setting.notify      Print a notice when the mask changes (true/false)\n\n" +
			"Examples:\n" +
			"  mask config setting.notify false\n" +
			"  mask config setting.automatic true\n" +
			"  mask config setting.notify",
		Args: cobra.RangeArgs(0, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadOrInit()
			if err != nil {
				return err
			}
			switch len(args) {
			case 0:
				for _, key := range settingKeys {
					v, _ := settingValue(cfg, key)
					fmt.Fprintf(cmd.OutOrStdout(), "%s = %t\n", key, v)
				}
				return nil
			case 1:
				v, err := settingValue(cfg, args[0])
				if err != nil {
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "%s = %t\n", args[0], v)
				return nil
			}
			key, value := args[0], args[1]
			v, err := parseSettingBool(value)
			if err != nil {
				return err
			}
			if err := setSetting(cfg, key, v); err != nil {
				return err
			}
			if err := cfg.Save(); err != nil {
				return err
			}
			fmt.Fprintf(cmd.ErrOrStderr(), "%s = %t\n", key, v)
			return nil
		},
	}
}

// settingKeys lists the settings accepted by `mask config`, in display order.
var settingKeys = []string{"setting.automatic", "setting.notify"}

// settingValue reads the named setting, reporting unknown keys.
func settingValue(cfg *config.Config, key string) (bool, error) {
	switch key {
	case "setting.automatic":
		return cfg.Automatic(), nil
	case "setting.notify":
		return cfg.Notify(), nil
	}
	return false, fmt.Errorf("unknown setting %q; available settings: %s", key, strings.Join(settingKeys, ", "))
}

// setSetting writes the named setting, reporting unknown keys.
func setSetting(cfg *config.Config, key string, v bool) error {
	switch key {
	case "setting.automatic":
		cfg.SetAutomatic(v)
	case "setting.notify":
		cfg.SetNotify(v)
	default:
		return fmt.Errorf("unknown setting %q; available settings: %s", key, strings.Join(settingKeys, ", "))
	}
	return nil
}

// parseSettingBool accepts the spellings people expect from settings commands.
func parseSettingBool(s string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "true", "on", "yes", "1":
		return true, nil
	case "false", "off", "no", "0":
		return false, nil
	}
	return false, fmt.Errorf("invalid value %q; use true or false", s)
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
				if rec, ok := st.Lookup(info.ID); ok {
					if rec.None {
						in.Cache = hookops.CacheNone
					} else {
						in.Cache = hookops.CacheMask
						in.CachedMask = rec.Mask
					}
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
			if d.Notice != "" && cfg.Notify() {
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

// rememberNone records that info was explicitly left without a mask.
func rememberNone(info repo.Info) error {
	st, err := state.Load()
	if err != nil {
		return err
	}
	st.RecordNone(info.ID, info.Root)
	return st.Save()
}

// maskChangeNotice describes a mask going on or off for the user.
func maskChangeNotice(verb, name string, info repo.Info, inRepo bool) string {
	if inRepo {
		return fmt.Sprintf("%s %q in %s", verb, name, info.Root)
	}
	return fmt.Sprintf("%s %q", verb, name)
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
			switch rec, ok := st.Lookup(info.ID); {
			case !ok:
				fmt.Fprintln(cmd.OutOrStdout(), "Remembered here: never set")
			case rec.None:
				fmt.Fprintln(cmd.OutOrStdout(), "Remembered here: none (mask off here)")
			default:
				fmt.Fprintf(cmd.OutOrStdout(), "Remembered here: %s\n", rec.Mask)
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
