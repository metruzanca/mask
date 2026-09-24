// Package maskops holds the decision logic behind the mask commands: building
// the environment payload for a mask, choosing the next mask on switch, and
// reporting what is currently worn. It is UI-free so it can be unit-tested.
package maskops

import (
	"fmt"
	"sort"

	"github.com/metruzanca/mask/internal/config"
	"github.com/metruzanca/mask/internal/shellout"
)

// BuildPayload turns a mask into the environment payload emitted on switch.
// The SSH key path is expanded before use.
func BuildPayload(name string, m config.Mask) shellout.Payload {
	key := config.ExpandPath(m.SSHKey)
	ssh := fmt.Sprintf("ssh -i %s -o IdentitiesOnly=yes", key)

	vars := []shellout.Assignment{
		{Name: "GIT_AUTHOR_NAME", Value: m.GitName},
		{Name: "GIT_AUTHOR_EMAIL", Value: m.GitEmail},
		{Name: "GIT_COMMITTER_NAME", Value: m.GitName},
		{Name: "GIT_COMMITTER_EMAIL", Value: m.GitEmail},
		{Name: "GIT_SSH_COMMAND", Value: ssh},
		{Name: "GIT_CONFIG_COUNT", Value: "3"},
		{Name: "GIT_CONFIG_KEY_0", Value: "user.name"},
		{Name: "GIT_CONFIG_VALUE_0", Value: m.GitName},
		{Name: "GIT_CONFIG_KEY_1", Value: "user.email"},
		{Name: "GIT_CONFIG_VALUE_1", Value: m.GitEmail},
		{Name: "GIT_CONFIG_KEY_2", Value: "core.sshCommand"},
		{Name: "GIT_CONFIG_VALUE_2", Value: ssh},
	}

	userNames := make([]string, 0, len(m.Env))
	for _, k := range sortedKeys(m.Env) {
		userNames = append(userNames, k)
		vars = append(vars, shellout.Assignment{Name: k, Value: m.Env[k]})
	}

	return shellout.Payload{Name: name, Vars: vars, UserEnvNames: userNames}
}

// OffPayload is the payload that clears the mask.
func OffPayload() shellout.Payload {
	return shellout.Payload{}
}

// SelectResult is the outcome of choosing a target mask.
type SelectResult struct {
	Name string
	// NeedUI is true when the caller must show an interactive dropdown
	// (three or more masks) because no target could be chosen in code.
	NeedUI bool
}

// Select chooses which mask `mask switch` should wear. With zero or one masks
// the answer is fixed; with two it toggles away from current; with three or
// more it reports NeedUI so the caller can show a dropdown.
func Select(cfg *config.Config, current, requested string) (SelectResult, error) {
	names := cfg.Names()
	if len(names) == 0 {
		return SelectResult{}, fmt.Errorf("no masks defined; run `mask create` first")
	}

	if requested != "" {
		if _, ok := cfg.Get(requested); !ok {
			return SelectResult{}, fmt.Errorf("unknown mask %q", requested)
		}
		return SelectResult{Name: requested}, nil
	}

	switch len(names) {
	case 1:
		return SelectResult{Name: names[0]}, nil
	case 2:
		for _, n := range names {
			if n != current {
				return SelectResult{Name: n}, nil
			}
		}
		return SelectResult{Name: names[0]}, nil
	default:
		return SelectResult{NeedUI: true}, nil
	}
}

// Report is what `mask` prints about the current shell.
type Report struct {
	Worn     bool
	Name     string
	GitName  string
	GitEmail string
}

// Describe builds the report for the given shell environment.
func Describe(current, gitName, gitEmail string) Report {
	return Report{
		Worn:     current != "",
		Name:     current,
		GitName:  gitName,
		GitEmail: gitEmail,
	}
}

// RenderReport formats a report for humans.
func RenderReport(r Report) string {
	if !r.Worn {
		out := "No mask worn.\n"
		if r.GitName != "" || r.GitEmail != "" {
			out += fmt.Sprintf("Git identity: %s <%s>\n", r.GitName, r.GitEmail)
		}
		out += "Run `mask switch` to put one on.\n"
		return out
	}
	out := fmt.Sprintf("Wearing mask: %s\n", r.Name)
	if r.GitName != "" || r.GitEmail != "" {
		out += fmt.Sprintf("Git identity: %s <%s>\n", r.GitName, r.GitEmail)
	} else {
		out += "Git identity: (unknown)\n"
	}
	return out
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// FormatEnvPreview is a helper for tests and listings: a stable "KEY=VALUE"
// rendering of a mask's user env vars.
func FormatEnvPreview(m config.Mask) []string {
	if len(m.Env) == 0 {
		return nil
	}
	out := make([]string, 0, len(m.Env))
	for _, k := range sortedKeys(m.Env) {
		out = append(out, k+"="+m.Env[k])
	}
	return out
}
