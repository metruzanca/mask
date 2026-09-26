package repo

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindDirectoryGit(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "proj")
	sub := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}

	got, ok := Find(sub)
	if !ok {
		t.Fatal("expected to find repo")
	}
	if got.Root != canonical(root) {
		t.Errorf("Root = %q, want %q", got.Root, canonical(root))
	}
	if got.GitDir != canonical(filepath.Join(root, ".git")) {
		t.Errorf("GitDir = %q", got.GitDir)
	}
}

func TestFindInnermostWins(t *testing.T) {
	dir := t.TempDir()
	outer := filepath.Join(dir, "outer")
	inner := filepath.Join(outer, "inner")
	if err := os.MkdirAll(filepath.Join(outer, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(inner, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	got, ok := Find(inner)
	if !ok {
		t.Fatal("expected repo")
	}
	if got.Root != canonical(inner) {
		t.Errorf("Root = %q, want inner %q", got.Root, canonical(inner))
	}

	got, ok = Find(outer)
	if !ok || got.Root != canonical(outer) {
		t.Errorf("outer Root = %q", got.Root)
	}
}

func TestFindGitFile(t *testing.T) {
	dir := t.TempDir()
	worktree := filepath.Join(dir, "wt")
	realdir := filepath.Join(dir, "realdir")
	if err := os.MkdirAll(worktree, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(realdir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(worktree, ".git"), []byte("gitdir: "+realdir+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, ok := Find(worktree)
	if !ok {
		t.Fatal("expected repo")
	}
	if got.Root != canonical(worktree) {
		t.Errorf("Root = %q, want %q", got.Root, canonical(worktree))
	}
	if got.GitDir != canonical(realdir) {
		t.Errorf("GitDir = %q, want %q", got.GitDir, canonical(realdir))
	}
}

func TestFindGitFileRelative(t *testing.T) {
	dir := t.TempDir()
	worktree := filepath.Join(dir, "wt")
	if err := os.MkdirAll(filepath.Join(worktree, ".gitdir", "x"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(worktree, ".git"), []byte("gitdir: .gitdir/x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, ok := Find(worktree)
	if !ok {
		t.Fatal("expected repo")
	}
	if got.GitDir != canonical(filepath.Join(worktree, ".gitdir", "x")) {
		t.Errorf("GitDir = %q", got.GitDir)
	}
}

func TestFindNone(t *testing.T) {
	dir := t.TempDir()
	if _, ok := Find(dir); ok {
		t.Error("expected no repo")
	}
}
