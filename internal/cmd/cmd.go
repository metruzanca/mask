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
	"github.com/metruzanca/mask/internal/maskops"
	"github.com/metruzanca/mask/internal/shellout"
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
	root.AddCommand(newVersion())
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
			out, err := shellout.Render(shell, maskops.BuildPayload(name, m))
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

func runStatus(cmd *cobra.Command) error {
	name := os.Getenv("GIT_AUTHOR_NAME")
	email := os.Getenv("GIT_AUTHOR_EMAIL")
	if name == "" && email == "" {
		name, email = globalGitIdentity()
	}
	report := maskops.Describe(os.Getenv("MASK_NAME"), name, email)
	_, err := io.WriteString(cmd.OutOrStdout(), maskops.RenderReport(report))
	return err
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
