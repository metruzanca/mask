// Package repo locates the git repository containing a directory. It walks up
// from the starting path and treats the presence and contents of a .git entry
// as the definitive answer, so nested repositories resolve to the innermost
// one and worktrees/submodules (where .git is a file) resolve to their real
// git directory.
package repo

import (
	"os"
	"path/filepath"
	"strings"
)

// Info describes the repository containing a directory.
type Info struct {
	// Root is the directory holding the .git entry (the work tree root).
	Root string
	// GitDir is the resolved git directory: the .git directory itself, or the
	// path named by a .git file for worktrees and submodules. It is
	// canonicalized, so it identifies the repository regardless of how the
	// work tree was reached.
	GitDir string
	// ID is the canonical repository identifier, equal to GitDir.
	ID string
}

// Find returns the repository containing dir. It returns false when dir is not
// inside a repository. Relative dirs are made absolute; symlinks in the paths
// that are returned are resolved when possible.
func Find(dir string) (Info, bool) {
	start, err := filepath.Abs(dir)
	if err != nil {
		return Info{}, false
	}
	if resolved, err := filepath.EvalSymlinks(start); err == nil {
		start = resolved
	}

	for d := start; ; {
		gitPath := filepath.Join(d, ".git")
		if info, err := os.Lstat(gitPath); err == nil {
			gitDir, ok := resolveGitDir(gitPath, info)
			if !ok {
				return Info{}, false
			}
			return Info{Root: d, GitDir: gitDir, ID: gitDir}, true
		}
		parent := filepath.Dir(d)
		if parent == d {
			return Info{}, false
		}
		d = parent
	}
}

// resolveGitDir turns a .git entry into a canonical git directory path.
func resolveGitDir(gitPath string, info os.FileInfo) (string, bool) {
	if info.IsDir() {
		return canonical(gitPath), true
	}
	target, err := readGitFile(gitPath)
	if err != nil {
		return "", false
	}
	if !filepath.IsAbs(target) {
		target = filepath.Join(filepath.Dir(gitPath), target)
	}
	return canonical(target), true
}

// readGitFile parses a .git file, which holds a single `gitdir: <path>` line.
func readGitFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		const prefix = "gitdir:"
		if !strings.HasPrefix(line, prefix) {
			continue
		}
		return strings.TrimSpace(strings.TrimPrefix(line, prefix)), nil
	}
	return "", os.ErrNotExist
}

// canonical returns an absolute, symlink-resolved path, falling back to the
// cleaned absolute path when resolution fails.
func canonical(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = filepath.Clean(path)
	}
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		return resolved
	}
	return abs
}
