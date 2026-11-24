package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/metruzanca/mask/internal/config"
)

func runMask(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	root := newRoot()
	root.SetArgs(args)
	var stdout, stderr bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	err := root.Execute()
	return stdout.String(), stderr.String(), err
}

func writeTestConfig(t *testing.T) {
	t.Helper()
	writeTestConfigWith(t, nil)
}

// writeTestConfigWith writes the standard two-mask config after applying tweak
// to the settings, if given.
func writeTestConfigWith(t *testing.T, tweak func(*config.Config)) {
	t.Helper()
	cfg := &config.Config{}
	cfg.Set("personal", config.Mask{GitName: "Personal", GitEmail: "personal@example.com"})
	cfg.Set("work", config.Mask{GitName: "Work", GitEmail: "work@example.com"})
	if tweak != nil {
		tweak(cfg)
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("save config: %v", err)
	}
}

func TestSwitchReportsWornMask(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("MASK_NAME", "")
	t.Setenv("MASK_REPO", "")
	t.Chdir(t.TempDir())
	writeTestConfig(t)

	stdout, stderr, err := runMask(t, "--shell=fish", "switch", "work")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stderr, `mask: wearing "work"`) {
		t.Errorf("stderr = %q, want switch notice", stderr)
	}
	if !strings.Contains(stdout, "set -gx MASK_NAME 'work'") {
		t.Errorf("stdout = %q, want shell code", stdout)
	}
}

func TestSwitchReportsRepo(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("MASK_NAME", "")
	t.Setenv("MASK_REPO", "")
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	writeTestConfig(t)

	_, stderr, err := runMask(t, "--shell=fish", "switch", "work")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stderr, `mask: wearing "work" in `) {
		t.Errorf("stderr = %q, want repo in switch notice", stderr)
	}
}

func TestOffReportsRemovedMask(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("MASK_NAME", "work")
	t.Setenv("MASK_REPO", "")
	t.Chdir(t.TempDir())
	writeTestConfig(t)

	stdout, stderr, err := runMask(t, "--shell=fish", "off")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stderr, `mask: took off "work"`) {
		t.Errorf("stderr = %q, want off notice", stderr)
	}
	if !strings.Contains(stdout, "set -e MASK_NAME") {
		t.Errorf("stdout = %q, want shell code", stdout)
	}
}

func TestOffSilentWhenUnmasked(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("MASK_NAME", "")
	t.Setenv("MASK_REPO", "")
	t.Chdir(t.TempDir())
	writeTestConfig(t)

	_, stderr, err := runMask(t, "--shell=fish", "off")
	if err != nil {
		t.Fatal(err)
	}
	if stderr != "" {
		t.Errorf("stderr = %q, want no notice when unmasked", stderr)
	}
}

func TestSwitchQuietWhenNotifyOff(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("MASK_NAME", "")
	t.Setenv("MASK_REPO", "")
	t.Chdir(t.TempDir())
	writeTestConfigWith(t, func(cfg *config.Config) { cfg.SetNotify(false) })

	stdout, stderr, err := runMask(t, "--shell=fish", "switch", "work")
	if err != nil {
		t.Fatal(err)
	}
	if stderr != "" {
		t.Errorf("stderr = %q, want no notice when notify is off", stderr)
	}
	if !strings.Contains(stdout, "set -gx MASK_NAME 'work'") {
		t.Errorf("stdout = %q, want shell code", stdout)
	}
}

func TestOffQuietWhenNotifyOff(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("MASK_NAME", "work")
	t.Setenv("MASK_REPO", "")
	t.Chdir(t.TempDir())
	writeTestConfigWith(t, func(cfg *config.Config) { cfg.SetNotify(false) })

	stdout, stderr, err := runMask(t, "--shell=fish", "off")
	if err != nil {
		t.Fatal(err)
	}
	if stderr != "" {
		t.Errorf("stderr = %q, want no notice when notify is off", stderr)
	}
	if !strings.Contains(stdout, "set -e MASK_NAME") {
		t.Errorf("stdout = %q, want shell code", stdout)
	}
}

func TestHookQuietWhenNotifyOff(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("MASK_NAME", "work")
	t.Setenv("MASK_REPO", "")
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	writeTestConfigWith(t, func(cfg *config.Config) { cfg.SetNotify(false) })

	stdout, stderr, err := runMask(t, "--shell=fish", "_hook")
	if err != nil {
		t.Fatal(err)
	}
	if stderr != "" {
		t.Errorf("stderr = %q, want no notice when notify is off", stderr)
	}
	if !strings.Contains(stdout, "set -e MASK_NAME") {
		t.Errorf("stdout = %q, want shell code", stdout)
	}
}

func TestConfigCreatesFileAndSetsNotify(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	stdout, stderr, err := runMask(t, "config", "setting.notify", "false")
	if err != nil {
		t.Fatal(err)
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want empty on set", stdout)
	}
	if !strings.Contains(stderr, "setting.notify = false") {
		t.Errorf("stderr = %q, want confirmation", stderr)
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load generated config: %v", err)
	}
	if cfg.Notify() {
		t.Error("setting.notify=false did not persist")
	}
}

func TestConfigShowsAllSettings(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	writeTestConfigWith(t, func(cfg *config.Config) { cfg.SetNotify(false) })

	stdout, _, err := runMask(t, "config")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"setting.automatic = true", "setting.notify = false"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("stdout = %q, want %q", stdout, want)
		}
	}
}

func TestConfigShowsOneSetting(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	writeTestConfig(t)

	stdout, _, err := runMask(t, "config", "setting.automatic")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(stdout) != "setting.automatic = true" {
		t.Errorf("stdout = %q, want automatic=true", stdout)
	}
}

func TestConfigRejectsUnknownSetting(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	_, _, err := runMask(t, "config", "setting.nope", "false")
	if err == nil {
		t.Fatal("want error for unknown setting")
	}
	if !strings.Contains(err.Error(), "unknown setting") {
		t.Errorf("err = %v, want unknown setting message", err)
	}
}

func TestConfigRejectsInvalidValue(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	_, _, err := runMask(t, "config", "setting.notify", "maybe")
	if err == nil {
		t.Fatal("want error for invalid value")
	}
	if !strings.Contains(err.Error(), "use true or false") {
		t.Errorf("err = %v, want value hint", err)
	}
}
