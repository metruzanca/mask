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
	cfg := &config.Config{}
	cfg.Set("personal", config.Mask{GitName: "Personal", GitEmail: "personal@example.com"})
	cfg.Set("work", config.Mask{GitName: "Work", GitEmail: "work@example.com"})
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
