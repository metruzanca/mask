package state

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSaveLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "repos.toml")
	s := New()
	s.Record("/id/one", "/root/one", "work")
	s.Record("/id/two", "/root/two", "personal")
	if err := s.SaveFile(path); err != nil {
		t.Fatal(err)
	}

	got, err := LoadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if m, ok := got.Lookup("/id/one"); !ok || m != "work" {
		t.Errorf("Lookup one = %q, %v", m, ok)
	}
	if ids := got.IDs(); len(ids) != 2 || ids[0] != "/id/one" {
		t.Errorf("IDs = %v", ids)
	}
}

func TestMissingFileIsEmpty(t *testing.T) {
	s, err := LoadFile(filepath.Join(t.TempDir(), "nope.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Repo) != 0 {
		t.Errorf("expected empty, got %v", s.Repo)
	}
}

func TestForget(t *testing.T) {
	s := New()
	s.Record("/id", "/root", "work")
	if !s.Forget("/id") {
		t.Fatal("Forget should report removal")
	}
	if s.Forget("/id") {
		t.Error("second Forget should report no removal")
	}
	if _, ok := s.Lookup("/id"); ok {
		t.Error("entry should be gone")
	}
}

func TestSavePermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "repos.toml")
	s := New()
	s.Record("/id", "/root", "work")
	if err := s.SaveFile(path); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("perm = %o, want 600", perm)
	}
}

func TestAppendLogTracksAndTrims(t *testing.T) {
	path := filepath.Join(t.TempDir(), "auto.log")
	for i := 0; i < maxLogLines+50; i++ {
		if err := AppendLogFile(path, "line"); err != nil {
			t.Fatal(err)
		}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Count(string(data), "\n")
	if lines > maxLogLines {
		t.Errorf("log has %d lines, want <= %d", lines, maxLogLines)
	}
	if !strings.Contains(string(data), " line") {
		t.Error("log line missing")
	}
}
