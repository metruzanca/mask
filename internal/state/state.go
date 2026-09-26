// Package state stores what mask has learned about repositories: the mask last
// worn in each repo, and a log of automatic changes. Both live next to the
// config file so users can version them (or not) with their dotfiles.
package state

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/BurntSushi/toml"

	"github.com/metruzanca/mask/internal/config"
)

const (
	reposFile = "repos.toml"
	logFile   = "auto.log"
	// maxLogLines bounds the append-only log; older lines are dropped.
	maxLogLines = 1000
)

// Repo is the remembered mask for one repository.
type Repo struct {
	Mask      string `toml:"mask"`
	Root      string `toml:"root"`
	UpdatedAt string `toml:"updated_at"`
}

// State is the repos.toml file.
type State struct {
	Repo map[string]Repo `toml:"repo"`
}

// Paths returns the repos store and log paths under the config directory.
func Paths() (reposPath, logPath string, err error) {
	dir, err := config.Dir()
	if err != nil {
		return "", "", err
	}
	return filepath.Join(dir, reposFile), filepath.Join(dir, logFile), nil
}

// New returns an empty state.
func New() *State { return &State{Repo: map[string]Repo{}} }

// Load reads the repos store. A missing file yields an empty store.
func Load() (*State, error) {
	path, _, err := Paths()
	if err != nil {
		return nil, err
	}
	return LoadFile(path)
}

// LoadFile reads a store from an explicit path.
func LoadFile(path string) (*State, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return New(), nil
		}
		return nil, err
	}
	s := &State{}
	if err := toml.Unmarshal(data, s); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	if s.Repo == nil {
		s.Repo = map[string]Repo{}
	}
	return s, nil
}

// Save writes the store atomically with tight permissions.
func (s *State) Save() error {
	path, _, err := Paths()
	if err != nil {
		return err
	}
	return s.SaveFile(path)
}

// SaveFile writes to an explicit path.
func (s *State) SaveFile(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	if s.Repo == nil {
		s.Repo = map[string]Repo{}
	}
	tmp := path + ".tmp"
	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	if err := toml.NewEncoder(f).Encode(s); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, path)
}

// Record remembers that id (a canonical repo id) wears name.
func (s *State) Record(id, root, name string) {
	if s.Repo == nil {
		s.Repo = map[string]Repo{}
	}
	s.Repo[id] = Repo{Mask: name, Root: root, UpdatedAt: time.Now().UTC().Format(time.RFC3339)}
}

// Lookup returns the remembered mask for id.
func (s *State) Lookup(id string) (string, bool) {
	r, ok := s.Repo[id]
	if !ok || r.Mask == "" {
		return "", false
	}
	return r.Mask, true
}

// Forget drops the entry for id. It returns whether anything was removed.
func (s *State) Forget(id string) bool {
	if _, ok := s.Repo[id]; !ok {
		return false
	}
	delete(s.Repo, id)
	return true
}

// IDs returns the stored repo ids in sorted order.
func (s *State) IDs() []string {
	ids := make([]string, 0, len(s.Repo))
	for id := range s.Repo {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// AppendLog adds a timestamped line to the automatic-change log, trimming the
// file to the most recent maxLogLines lines. The line is prefixed with an
// RFC3339 UTC timestamp.
func AppendLog(line string) error {
	_, path, err := Paths()
	if err != nil {
		return err
	}
	return AppendLogFile(path, line)
}

// AppendLogFile appends to an explicit log path (used by tests).
func AppendLogFile(path, line string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	existing := trimLines(readLines(path), maxLogLines-1)
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	w := bufio.NewWriter(f)
	for _, l := range existing {
		fmt.Fprintln(w, l)
	}
	fmt.Fprintf(w, "%s %s\n", time.Now().UTC().Format(time.RFC3339), line)
	if err := w.Flush(); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// readLines returns the non-empty lines of a file, or nil when it is missing.
func readLines(path string) []string {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	var lines []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	return lines
}

// trimLines keeps at most the last n lines.
func trimLines(lines []string, n int) []string {
	if n < 0 {
		n = 0
	}
	if len(lines) <= n {
		return lines
	}
	return lines[len(lines)-n:]
}
