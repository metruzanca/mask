package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mask", "config.toml")

	cfg := &Config{}
	cfg.Set("personal", Mask{
		GitName:  "metru.dev",
		GitEmail: "metru@zanca.dev",
		SSHKey:   "~/.ssh/id_ed25519",
		Env:      map[string]string{"GITHUB_TOKEN": "ghp_x", "AWS_PROFILE": "personal"},
	})
	cfg.Set("work", Mask{
		GitName:  "Metru Zanca",
		GitEmail: "metru@zanca.dev",
		SSHKey:   "~/.ssh/id_ed25519_terminal_shop",
	})

	if err := cfg.SaveFile(path); err != nil {
		t.Fatalf("SaveFile: %v", err)
	}

	got, err := LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	if len(got.Masks) != 2 {
		t.Fatalf("got %d masks, want 2", len(got.Masks))
	}
	p, ok := got.Get("personal")
	if !ok {
		t.Fatal("personal missing")
	}
	if p.GitName != "metru.dev" || p.GitEmail != "metru@zanca.dev" {
		t.Errorf("identity mismatch: %+v", p)
	}
	if p.Env["AWS_PROFILE"] != "personal" {
		t.Errorf("env mismatch: %+v", p.Env)
	}
	if names := got.Names(); names[0] != "personal" || names[1] != "work" {
		t.Errorf("Names not sorted: %v", names)
	}
}

func TestLoadMissing(t *testing.T) {
	_, err := LoadFile(filepath.Join(t.TempDir(), "nope.toml"))
	if !errors.Is(err, ErrNotExist) {
		t.Fatalf("want ErrNotExist, got %v", err)
	}
}

func TestSavePermissions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	cfg := &Config{}
	cfg.Set("x", Mask{GitName: "X"})
	if err := cfg.SaveFile(path); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("file perm = %o, want 600", perm)
	}
}

func TestExpandPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home dir")
	}
	cases := map[string]string{
		"~/.ssh/id_ed25519": filepath.Join(home, ".ssh/id_ed25519"),
		"~":                 home,
		"/absolute/path":    "/absolute/path",
		"relative/path":     "relative/path",
	}
	for in, want := range cases {
		if got := ExpandPath(in); got != want {
			t.Errorf("ExpandPath(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestPathHonorsXDG(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/tmp/xdg-test")
	got, err := Path()
	if err != nil {
		t.Fatal(err)
	}
	if got != "/tmp/xdg-test/mask/config.toml" {
		t.Errorf("Path() = %q", got)
	}
}
