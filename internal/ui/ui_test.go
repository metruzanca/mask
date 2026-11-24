package ui

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultKeyPath(t *testing.T) {
	cases := []struct {
		name string
		want string
	}{
		{"work", "~/.ssh/id_ed25519_work"},
		{"Work Stuff!", "~/.ssh/id_ed25519_work-stuff"},
		{"  work   site  ", "~/.ssh/id_ed25519_work-site"},
		{"personal_2", "~/.ssh/id_ed25519_personal_2"},
		{"", "~/.ssh/id_ed25519_mask"},
		{"---", "~/.ssh/id_ed25519_mask"},
	}
	for _, tc := range cases {
		if got := defaultKeyPath(tc.name); got != tc.want {
			t.Errorf("defaultKeyPath(%q) = %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestCheckNewKeyPath(t *testing.T) {
	dir := t.TempDir()
	private := filepath.Join(dir, "id_ed25519_test")

	if err := checkNewKeyPath(private); err != nil {
		t.Fatalf("fresh path rejected: %v", err)
	}
	if err := checkNewKeyPath(private + ".pub"); err == nil {
		t.Fatal("path ending in .pub accepted")
	}

	if err := os.WriteFile(private, []byte("key"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := checkNewKeyPath(private); err == nil {
		t.Fatal("existing private key accepted")
	}
	if err := os.Remove(private); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(private+".pub", []byte("key"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := checkNewKeyPath(private); err == nil {
		t.Fatal("existing public key accepted")
	}
}

func TestGenerateSSHKey(t *testing.T) {
	requireSSHKeygen(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "id_ed25519_test")

	if err := GenerateSSHKey(path, "test@example.com", ""); err != nil {
		t.Fatalf("GenerateSSHKey: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("private key missing: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("private key mode = %o, want 600", perm)
	}
	pub, err := os.ReadFile(path + ".pub")
	if err != nil {
		t.Fatalf("public key missing: %v", err)
	}
	if !strings.Contains(string(pub), "test@example.com") {
		t.Errorf("public key lacks comment: %q", pub)
	}
	if err := GenerateSSHKey(path, "test@example.com", ""); err == nil {
		t.Fatal("generating over an existing key succeeded")
	}
}

func TestGenerateSSHKeyWithPassphrase(t *testing.T) {
	requireSSHKeygen(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "id_ed25519_pass")

	if err := GenerateSSHKey(path, "test@example.com", "correct horse battery staple"); err != nil {
		t.Fatalf("GenerateSSHKey: %v", err)
	}
	out, err := exec.Command("ssh-keygen", "-y", "-P", "correct horse battery staple", "-f", path).CombinedOutput()
	if err != nil {
		t.Fatalf("generated key does not unlock with passphrase: %v\n%s", err, out)
	}
	if !strings.HasPrefix(strings.TrimSpace(string(out)), "ssh-ed25519 ") {
		t.Errorf("unexpected public key output: %q", out)
	}
}

func requireSSHKeygen(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("ssh-keygen"); err != nil {
		t.Skip("ssh-keygen not installed")
	}
}
