// Package shellout renders the shell code that `mask switch` and `mask off`
// emit on stdout. The user's shell function pipes that stdout into `source`,
// which is the only way a CLI can change its parent shell's environment.
//
// Everything is injected through the environment rather than written to
// ~/.gitconfig, which is often a read-only Nix/home-manager symlink. Git's
// GIT_CONFIG_* runtime config and GIT_SSH_COMMAND are verified to override
// global config for commits and pushes.
package shellout

import (
	"fmt"
	"sort"
	"strings"
)

// Shell is a supported shell family.
type Shell string

const (
	Fish Shell = "fish"
	Bash Shell = "bash"
	Zsh  Shell = "zsh"
)

// Supported returns the shell families the integration can generate.
func Supported() []Shell { return []Shell{Fish, Bash, Zsh} }

// Parse normalizes a shell argument (e.g. /bin/zsh, ZSH) to a Shell.
func Parse(s string) (Shell, bool) {
	n := strings.ToLower(strings.TrimSpace(s))
	n = strings.Trim(n, "/")
	if i := strings.LastIndexByte(n, '/'); i >= 0 {
		n = n[i+1:]
	}
	switch n {
	case "fish":
		return Fish, true
	case "bash", "sh":
		return Bash, true
	case "zsh":
		return Zsh, true
	}
	return "", false
}

// managedVars are the fixed environment variables mask owns. They are always
// cleared before a new mask is applied so no stale value leaks between masks.
var managedVars = []string{
	"MASK_NAME",
	"MASK_ENV_MASK",
	"GIT_AUTHOR_NAME",
	"GIT_AUTHOR_EMAIL",
	"GIT_COMMITTER_NAME",
	"GIT_COMMITTER_EMAIL",
	"GIT_SSH_COMMAND",
	"GIT_CONFIG_COUNT",
	"GIT_CONFIG_KEY_0",
	"GIT_CONFIG_KEY_1",
	"GIT_CONFIG_KEY_2",
	"GIT_CONFIG_VALUE_0",
	"GIT_CONFIG_VALUE_1",
	"GIT_CONFIG_VALUE_2",
}

// Assignment is one environment variable to export.
type Assignment struct {
	Name  string
	Value string
}

// Payload describes the environment change to emit.
type Payload struct {
	// Name is the mask name written to MASK_NAME, or "" to turn mask off.
	Name string
	// Vars are the values to export for the mask (git identity, SSH command,
	// and the user's own env vars).
	Vars []Assignment
	// UserEnvNames are the user-defined variable names, tracked in
	// MASK_ENV_MASK so the next switch can unset them.
	UserEnvNames []string
}

// Render returns shell code that unsets the previous mask's variables and
// applies p. The output is deterministic for a given payload.
func Render(shell Shell, p Payload) (string, error) {
	switch shell {
	case Fish:
		return fishPayload(p), nil
	case Bash, Zsh:
		return shPayload(p), nil
	}
	return "", fmt.Errorf("unsupported shell %q", shell)
}

// Init returns the wrapper function that must be installed once, e.g. via
// `mask init fish | source`. It makes `mask switch` and `mask off` source the
// binary's stdout, while every other subcommand runs untouched. The wrapper
// tells the binary which shell it is, because $SHELL names the login shell,
// not necessarily the shell currently running.
func Init(shell Shell) (string, error) {
	switch shell {
	case Fish:
		return fishInit, nil
	case Bash, Zsh:
		return shInit(string(shell)), nil
	}
	return "", fmt.Errorf("unsupported shell %q", shell)
}

// ConfigPath returns the shell's startup file, for setup instructions.
func ConfigPath(shell Shell) string {
	switch shell {
	case Fish:
		return "~/.config/fish/config.fish"
	case Zsh:
		return "~/.zshrc"
	default:
		return "~/.bashrc"
	}
}

// SetupCommand returns the line a user appends to their shell config to load
// the wrapper on startup.
func SetupCommand(shell Shell) string {
	switch shell {
	case Fish:
		return "mask init fish | source"
	case Zsh:
		return `eval "$(mask init zsh)"`
	default:
		return `eval "$(mask init bash)"`
	}
}

// DetectShell guesses the caller's shell from the SHELL env var, returning ""
// when it cannot tell. The lookup is injected so tests stay hermetic.
func DetectShell(env func(string) string) (Shell, bool) {
	base := env("SHELL")
	if i := strings.LastIndexByte(base, '/'); i >= 0 {
		base = base[i+1:]
	}
	switch {
	case strings.Contains(base, "fish"):
		return Fish, true
	case strings.Contains(base, "zsh"):
		return Zsh, true
	case strings.Contains(base, "bash"), base == "sh":
		return Bash, true
	}
	switch {
	case env("ZSH_VERSION") != "":
		return Zsh, true
	case env("BASH_VERSION") != "":
		return Bash, true
	}
	return "", false
}

const fishInit = `# mask init fish (generated, do not edit)
function mask
    switch "$argv[1]"
        case switch off
            command mask --shell=fish $argv | source
        case '*'
            command mask $argv
    end
end
`

func shInit(shell string) string {
	return `# mask init ` + shell + ` (generated, do not edit)
mask() {
    case "$1" in
        switch|off) eval "$(command mask --shell=` + shell + ` "$@")" ;;
        *) command mask "$@" ;;
    esac
}
`
}

func fishPayload(p Payload) string {
	var b strings.Builder
	b.WriteString("# mask: clear previous environment\n")
	b.WriteString("if set -q MASK_ENV_MASK\n")
	b.WriteString("    for _mask_v in $MASK_ENV_MASK\n")
	b.WriteString("        set -e $_mask_v\n")
	b.WriteString("    end\n")
	b.WriteString("end\n")
	b.WriteString("set -e _mask_v\n")
	for _, name := range managedVars {
		fmt.Fprintf(&b, "set -e %s\n", name)
	}
	b.WriteString("# mask: apply environment\n")
	if p.Name != "" {
		fmt.Fprintf(&b, "set -gx MASK_NAME %s\n", fishQuote(p.Name))
	}
	for _, a := range p.Vars {
		fmt.Fprintf(&b, "set -gx %s %s\n", a.Name, fishQuote(a.Value))
	}
	if len(p.UserEnvNames) > 0 {
		names := append([]string(nil), p.UserEnvNames...)
		sort.Strings(names)
		quoted := make([]string, len(names))
		for i, n := range names {
			quoted[i] = fishQuote(n)
		}
		fmt.Fprintf(&b, "set -gx MASK_ENV_MASK %s\n", strings.Join(quoted, " "))
	}
	return b.String()
}

func shPayload(p Payload) string {
	var b strings.Builder
	b.WriteString("# mask: clear previous environment\n")
	b.WriteString(`if [ -n "${MASK_ENV_MASK:-}" ]; then` + "\n")
	b.WriteString("    for _mask_v in $MASK_ENV_MASK; do unset \"$_mask_v\"; done\n")
	b.WriteString("fi\n")
	b.WriteString("unset _mask_v\n")
	for _, name := range managedVars {
		fmt.Fprintf(&b, "unset %s\n", name)
	}
	b.WriteString("# mask: apply environment\n")
	if p.Name != "" {
		fmt.Fprintf(&b, "export MASK_NAME=%s\n", shQuote(p.Name))
	}
	for _, a := range p.Vars {
		fmt.Fprintf(&b, "export %s=%s\n", a.Name, shQuote(a.Value))
	}
	if len(p.UserEnvNames) > 0 {
		names := append([]string(nil), p.UserEnvNames...)
		sort.Strings(names)
		fmt.Fprintf(&b, "export MASK_ENV_MASK=%s\n", shQuote(strings.Join(names, " ")))
	}
	return b.String()
}

// fishQuote wraps s so fish reads it literally. In single quotes fish only
// treats \ and ' specially, escaping both matches `string escape` output.
func fishQuote(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `'`, `\'`)
	return "'" + s + "'"
}

// shQuote wraps s in POSIX single quotes, escaping embedded quotes.
func shQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
