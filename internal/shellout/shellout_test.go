package shellout

import (
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	cases := map[string]Shell{
		"fish":          Fish,
		"/bin/zsh":      Zsh,
		"ZSH":           Zsh,
		"/usr/bin/bash": Bash,
		"sh":            Bash,
	}
	for in, want := range cases {
		got, ok := Parse(in)
		if !ok || got != want {
			t.Errorf("Parse(%q) = %q, %v; want %q", in, got, ok, want)
		}
	}
	if _, ok := Parse("powershell"); ok {
		t.Error("Parse(powershell) should fail")
	}
}

func TestFishQuote(t *testing.T) {
	cases := map[string]string{
		"simple":        "'simple'",
		"with space":    "'with space'",
		"it's":          `'it\'s'`,
		`back\slash`:    `'back\\slash'`,
		"$HOME`whoami`": "'$HOME`whoami`'",
	}
	for in, want := range cases {
		if got := fishQuote(in); got != want {
			t.Errorf("fishQuote(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestShQuote(t *testing.T) {
	cases := map[string]string{
		"simple": "'simple'",
		"it's":   `'it'\''s'`,
		"$HOME":  "'$HOME'",
		"a b":    "'a b'",
	}
	for in, want := range cases {
		if got := shQuote(in); got != want {
			t.Errorf("shQuote(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestRenderFishClearsThenApplies(t *testing.T) {
	p := Payload{
		Name: "personal",
		Vars: []Assignment{
			{Name: "GIT_AUTHOR_NAME", Value: "metru.dev"},
			{Name: "GIT_SSH_COMMAND", Value: "ssh -i /k -o IdentitiesOnly=yes"},
		},
		UserEnvNames: []string{"AWS_PROFILE", "GITHUB_TOKEN"},
	}
	out, err := Render(Fish, p)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"set -e GIT_AUTHOR_NAME",
		"set -gx MASK_NAME 'personal'",
		"set -gx GIT_AUTHOR_NAME 'metru.dev'",
		"set -gx MASK_ENV_MASK 'AWS_PROFILE' 'GITHUB_TOKEN'",
		"for _mask_v in $MASK_ENV_MASK",
		"set -e $_mask_v",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("fish output missing %q\n%s", want, out)
		}
	}
	// MASK_ENV_MASK must be a real fish list (space-separated quoted words),
	// not one single quoted string.
	if strings.Contains(out, `MASK_ENV_MASK 'AWS_PROFILE GITHUB_TOKEN'`) {
		t.Errorf("MASK_ENV_MASK rendered as a single string, want a list:\n%s", out)
	}
}

func TestRenderShClearsThenApplies(t *testing.T) {
	p := Payload{
		Name:         "work",
		Vars:         []Assignment{{Name: "GIT_AUTHOR_EMAIL", Value: "w@x.com"}},
		UserEnvNames: []string{"TOKEN"},
	}
	out, err := Render(Bash, p)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`if [ -n "${MASK_ENV_MASK:-}" ]; then`,
		`for _mask_v in $MASK_ENV_MASK; do unset "$_mask_v"; done`,
		"unset GIT_AUTHOR_EMAIL",
		"export MASK_NAME='work'",
		"export GIT_AUTHOR_EMAIL='w@x.com'",
		"export MASK_ENV_MASK='TOKEN'",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("sh output missing %q\n%s", want, out)
		}
	}
}

func TestOffPayloadClearsEverything(t *testing.T) {
	out, err := Render(Fish, Payload{})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "set -gx MASK_NAME") {
		t.Errorf("off should not set MASK_NAME:\n%s", out)
	}
	for _, name := range managedVars {
		if !strings.Contains(out, "set -e "+name) {
			t.Errorf("off missing clear of %s", name)
		}
	}
}

func TestInitDeclaresShell(t *testing.T) {
	fishOut, _ := Init(Fish)
	if !strings.Contains(fishOut, "command mask --shell=fish $argv | source") {
		t.Errorf("fish init must declare --shell=fish:\n%s", fishOut)
	}
	bashOut, _ := Init(Bash)
	if !strings.Contains(bashOut, `command mask --shell=bash "$@"`) {
		t.Errorf("bash init must declare --shell=bash:\n%s", bashOut)
	}
	zshOut, _ := Init(Zsh)
	if !strings.Contains(zshOut, `command mask --shell=zsh "$@"`) {
		t.Errorf("zsh init must declare --shell=zsh:\n%s", zshOut)
	}
	if !strings.Contains(fishOut, "case switch off") {
		t.Errorf("fish init should source only switch/off:\n%s", fishOut)
	}
	if !strings.Contains(bashOut, "switch|off)") {
		t.Errorf("bash init should source only switch/off:\n%s", bashOut)
	}
}

func TestDetectShell(t *testing.T) {
	env := func(m map[string]string) func(string) string {
		return func(k string) string { return m[k] }
	}
	if s, ok := DetectShell(env(map[string]string{"SHELL": "/run/current-system/sw/bin/fish"})); !ok || s != Fish {
		t.Errorf("fish detection failed: %q %v", s, ok)
	}
	if s, ok := DetectShell(env(map[string]string{"BASH_VERSION": "5.2"})); !ok || s != Bash {
		t.Errorf("bash detection failed: %q %v", s, ok)
	}
	if _, ok := DetectShell(env(map[string]string{})); ok {
		t.Error("empty env should not detect a shell")
	}
}
