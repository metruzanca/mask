// Package ui builds the charmbracelet/huh forms used by mask. Forms render to
// stderr (huh's default), which keeps them off the stdout that the mask shell
// function sources, and keeps typed values out of the shell's history.
package ui

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/huh"
)

// CreateResult holds the values collected by the create form.
type CreateResult struct {
	Name     string
	GitName  string
	GitEmail string
	SSHKey   string
}

// CreateForm asks for a new mask's name, git identity, and SSH key.
func CreateForm() (CreateResult, error) {
	var res CreateResult

	keys := SSHKeys()
	keyField := huh.NewInput().
		Title("SSH key").
		Description("Path to the private key used for git push").
		Placeholder("~/.ssh/id_ed25519").
		Value(&res.SSHKey)

	var keyGroup *huh.Group
	if len(keys) > 0 {
		res.SSHKey = keys[0]
		keyGroup = huh.NewGroup(
			huh.NewSelect[string]().
				Title("SSH key").
				Description("Which key should git push use?").
				Options(huh.NewOptions(keys...)...).
				Value(&res.SSHKey),
		)
	} else {
		keyGroup = huh.NewGroup(keyField)
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Mask name").
				Description("A label for this identity, used by `mask switch`").
				Placeholder("personal").
				Value(&res.Name).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return errEmpty("mask name")
					}
					return nil
				}),
			huh.NewInput().
				Title("Git user.name").
				Placeholder("Jane Doe").
				Value(&res.GitName).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return errEmpty("git user.name")
					}
					return nil
				}),
			huh.NewInput().
				Title("Git user.email").
				Placeholder("jane@example.com").
				Value(&res.GitEmail).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return errEmpty("git user.email")
					}
					return nil
				}),
		),
		keyGroup,
	)

	if err := form.Run(); err != nil {
		return CreateResult{}, err
	}
	res.Name = strings.TrimSpace(res.Name)
	res.GitName = strings.TrimSpace(res.GitName)
	res.GitEmail = strings.TrimSpace(res.GitEmail)
	res.SSHKey = strings.TrimSpace(res.SSHKey)
	return res, nil
}

// SwitchForm shows a dropdown of mask names and returns the chosen one.
func SwitchForm(names []string) (string, error) {
	if len(names) == 0 {
		return "", errEmpty("mask")
	}
	choice := names[0]
	err := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Wear which mask?").
				Options(huh.NewOptions(names...)...).
				Value(&choice),
		),
	).Run()
	if err != nil {
		return "", err
	}
	return choice, nil
}

// SSHKeys returns candidate private keys in ~/.ssh, sorted, as display paths.
func SSHKeys() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	dir := filepath.Join(home, ".ssh")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var keys []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(name, ".pub") || strings.HasPrefix(name, ".") {
			continue
		}
		switch name {
		case "config", "known_hosts", "known_hosts.old", "authorized_keys", "environment", "rc":
			continue
		}
		keys = append(keys, "~/.ssh/"+name)
	}
	sort.Strings(keys)
	return keys
}

type errEmpty string

func (e errEmpty) Error() string { return string(e) + " is required" }
