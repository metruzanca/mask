// Package config loads and saves the mask configuration file at
// ~/.config/mask/config.toml. A config holds named masks, each with the git
// identity and SSH key to wear, plus optional environment variables.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
)

// Mask is one named identity.
type Mask struct {
	GitName  string            `toml:"git_name"`
	GitEmail string            `toml:"git_email"`
	SSHKey   string            `toml:"ssh_key"`
	Env      map[string]string `toml:"env,omitempty"`
}

// Settings is the [settings] table. Automatic is a pointer so an absent field
// can default to on while an explicit `automatic = false` is honored.
type Settings struct {
	Automatic *bool `toml:"automatic,omitempty"`
}

// Config is the whole config file.
type Config struct {
	Masks    map[string]Mask `toml:"mask"`
	Settings Settings        `toml:"settings,omitempty"`
}

// Automatic reports whether per-repo automatic masking is enabled. It defaults
// to true when the setting is absent.
func (c *Config) Automatic() bool {
	if c.Settings.Automatic == nil {
		return true
	}
	return *c.Settings.Automatic
}

// SetAutomatic writes the automatic setting.
func (c *Config) SetAutomatic(v bool) {
	c.Settings.Automatic = &v
}

// ErrNotExist is returned by Load when there is no config file yet.
var ErrNotExist = errors.New("no config file")

// Dir returns the mask config directory, honoring XDG_CONFIG_HOME.
func Dir() (string, error) {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "mask"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "mask"), nil
}

// Path returns the config file path, honoring XDG_CONFIG_HOME.
func Path() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.toml"), nil
}

// Load reads the config file. A missing file yields an empty config and an
// error wrapping ErrNotExist, so callers can distinguish "first run".
func Load() (*Config, error) {
	path, err := Path()
	if err != nil {
		return nil, err
	}
	return LoadFile(path)
}

// LoadFile reads a config from an explicit path (used by tests).
func LoadFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &Config{Masks: map[string]Mask{}}, fmt.Errorf("%w: %s", ErrNotExist, path)
		}
		return nil, err
	}
	cfg := &Config{}
	if err := toml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	if cfg.Masks == nil {
		cfg.Masks = map[string]Mask{}
	}
	return cfg, nil
}

// Save writes the config atomically with tight permissions.
func (c *Config) Save() error {
	path, err := Path()
	if err != nil {
		return err
	}
	return c.SaveFile(path)
}

// SaveFile writes to an explicit path (used by tests).
func (c *Config) SaveFile(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	if c.Masks == nil {
		c.Masks = map[string]Mask{}
	}
	tmp := path + ".tmp"
	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	if err := toml.NewEncoder(f).Encode(c); err != nil {
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

// Names returns the configured mask names in sorted order.
func (c *Config) Names() []string {
	names := make([]string, 0, len(c.Masks))
	for name := range c.Masks {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Get returns the named mask.
func (c *Config) Get(name string) (Mask, bool) {
	m, ok := c.Masks[name]
	return m, ok
}

// Set stores a mask under name.
func (c *Config) Set(name string, m Mask) {
	if c.Masks == nil {
		c.Masks = map[string]Mask{}
	}
	c.Masks[name] = m
}

// ExpandPath expands a leading ~ to the user's home directory. Other paths
// are returned unchanged.
func ExpandPath(p string) string {
	if p == "~" || strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			if p == "~" {
				return home
			}
			return filepath.Join(home, p[2:])
		}
	}
	return p
}
