package maskops

import (
	"strings"
	"testing"

	"github.com/metruzanca/mask/internal/config"
)

func cfgWith(names ...string) *config.Config {
	c := &config.Config{}
	for _, n := range names {
		c.Set(n, config.Mask{GitName: strings.ToUpper(n), GitEmail: n + "@x.com"})
	}
	return c
}

func TestSelectToggleWithTwo(t *testing.T) {
	cfg := cfgWith("a", "b")
	got, err := Select(cfg, "a", "")
	if err != nil {
		t.Fatal(err)
	}
	if got.NeedUI || got.Name != "b" {
		t.Errorf("current=a: got %+v, want b", got)
	}
	got, _ = Select(cfg, "b", "")
	if got.Name != "a" {
		t.Errorf("current=b: got %+v, want a", got)
	}
	got, _ = Select(cfg, "", "")
	if got.Name != "a" {
		t.Errorf("no current: got %+v, want a (first)", got)
	}
}

func TestSelectSingleAutoPicks(t *testing.T) {
	got, err := Select(cfgWith("only"), "", "")
	if err != nil {
		t.Fatal(err)
	}
	if got.NeedUI || got.Name != "only" {
		t.Errorf("got %+v, want only", got)
	}
}

func TestSelectThreeNeedsUI(t *testing.T) {
	got, err := Select(cfgWith("a", "b", "c"), "a", "")
	if err != nil {
		t.Fatal(err)
	}
	if !got.NeedUI {
		t.Errorf("three masks should need UI, got %+v", got)
	}
}

func TestSelectExplicitAndUnknown(t *testing.T) {
	cfg := cfgWith("a", "b", "c")
	got, err := Select(cfg, "a", "c")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "c" {
		t.Errorf("explicit got %+v, want c", got)
	}
	if _, err := Select(cfg, "a", "nope"); err == nil {
		t.Error("unknown name should error")
	}
	if _, err := Select(&config.Config{}, "", ""); err == nil {
		t.Error("empty config should error")
	}
}

func TestBuildPayload(t *testing.T) {
	m := config.Mask{
		GitName:  "Metru Zanca",
		GitEmail: "metru@zanca.dev",
		SSHKey:   "~/.ssh/id_ed25519_terminal_shop",
		Env: map[string]string{
			"AWS_PROFILE":  "work",
			"GITHUB_TOKEN": "ghp_1",
		},
	}
	p := BuildPayload("work", m)
	if p.Name != "work" {
		t.Errorf("Name = %q", p.Name)
	}
	vals := map[string]string{}
	for _, a := range p.Vars {
		vals[a.Name] = a.Value
	}
	if vals["GIT_AUTHOR_NAME"] != "Metru Zanca" || vals["GIT_COMMITTER_EMAIL"] != "metru@zanca.dev" {
		t.Errorf("identity vars wrong: %+v", vals)
	}
	if vals["GIT_CONFIG_COUNT"] != "3" {
		t.Errorf("GIT_CONFIG_COUNT = %q, want 3", vals["GIT_CONFIG_COUNT"])
	}
	if vals["GIT_CONFIG_KEY_0"] != "user.name" || vals["GIT_CONFIG_KEY_2"] != "core.sshCommand" {
		t.Errorf("config keys wrong: %+v", vals)
	}
	if !strings.Contains(vals["GIT_SSH_COMMAND"], "-o IdentitiesOnly=yes") {
		t.Errorf("ssh command lacks IdentitiesOnly: %q", vals["GIT_SSH_COMMAND"])
	}
	if !strings.Contains(vals["GIT_SSH_COMMAND"], string(config.ExpandPath(m.SSHKey))) {
		t.Errorf("ssh command not expanded: %q", vals["GIT_SSH_COMMAND"])
	}
	if len(p.UserEnvNames) != 2 || p.UserEnvNames[0] != "AWS_PROFILE" || p.UserEnvNames[1] != "GITHUB_TOKEN" {
		t.Errorf("UserEnvNames = %v, want sorted [AWS_PROFILE GITHUB_TOKEN]", p.UserEnvNames)
	}
	// core.sshCommand config value must equal GIT_SSH_COMMAND.
	if vals["GIT_CONFIG_VALUE_2"] != vals["GIT_SSH_COMMAND"] {
		t.Errorf("core.sshCommand mismatch: %q vs %q", vals["GIT_CONFIG_VALUE_2"], vals["GIT_SSH_COMMAND"])
	}
}

func TestRenderReport(t *testing.T) {
	worn := RenderReport(Describe("personal", "metru.dev", "metru@zanca.dev"))
	if !strings.Contains(worn, "Wearing mask: personal") || !strings.Contains(worn, "metru.dev <metru@zanca.dev>") {
		t.Errorf("worn report wrong:\n%s", worn)
	}
	off := RenderReport(Describe("", "metru.dev", "metru@zanca.dev"))
	if !strings.Contains(off, "No mask worn.") || !strings.Contains(off, "metru.dev") {
		t.Errorf("off report wrong:\n%s", off)
	}
}
