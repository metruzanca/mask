// Package ui builds the charmbracelet/huh forms used by mask. Forms render to
// stderr (huh's default), which keeps them off the stdout that the mask shell
// function sources, and keeps typed values out of the shell's history.
package ui

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/huh"

	"github.com/metruzanca/mask/internal/config"
)

// CreateResult holds the values collected by the create form.
type CreateResult struct {
	Name     string
	GitName  string
	GitEmail string
	SSHKey   string

	// NewKey is true when SSHKey should be generated before saving the mask.
	NewKey bool
	// NewKeyPath is the display path (not expanded) for the generated key.
	NewKeyPath string
	// Passphrase protects the generated key. It is never written to config.
	Passphrase string
}

const (
	keyChoiceManual = "\x00mask-manual-key"
	keyChoiceCreate = "\x00mask-create-key"
)

// CreateForm asks for a new mask's name, git identity, and SSH key.
func CreateForm() (CreateResult, error) {
	var res CreateResult
	var confirmPassphrase string

	keys := SSHKeys()
	keyChoice := keyChoiceCreate
	if len(keys) > 0 {
		keyChoice = keys[0]
	}

	keyOptions := make([]huh.Option[string], 0, len(keys)+2)
	for _, k := range keys {
		keyOptions = append(keyOptions, huh.NewOption(k, k))
	}
	keyOptions = append(keyOptions,
		huh.NewOption("Enter a path manually…", keyChoiceManual),
		huh.NewOption("Create a new SSH key…", keyChoiceCreate),
	)

	keyGroup := huh.NewGroup(
		huh.NewSelect[string]().
			Title("SSH key").
			Description("Which key should git push use?").
			Options(keyOptions...).
			Value(&keyChoice),
	)

	manualGroup := huh.NewGroup(
		huh.NewInput().
			Title("SSH key path").
			Description("Path to the private key used for git push").
			Placeholder("~/.ssh/id_ed25519").
			Value(&res.SSHKey).
			Validate(func(s string) error {
				if strings.TrimSpace(s) == "" {
					return errEmpty("SSH key path")
				}
				return nil
			}),
	).WithHideFunc(func() bool { return keyChoice != keyChoiceManual })

	createGroup := huh.NewGroup(
		huh.NewInput().
			Title("New key path").
			Description("Leave blank to use ~/.ssh/id_ed25519_<mask name>").
			Placeholder("~/.ssh/id_ed25519_<mask name>").
			Value(&res.NewKeyPath).
			Validate(func(s string) error {
				path := strings.TrimSpace(s)
				if path == "" {
					path = defaultKeyPath(res.Name)
				}
				return checkNewKeyPath(path)
			}),
		huh.NewInput().
			Title("Passphrase (optional)").
			Description("Leave blank for no passphrase").
			EchoMode(huh.EchoModePassword).
			Value(&res.Passphrase),
		huh.NewInput().
			Title("Confirm passphrase").
			EchoMode(huh.EchoModePassword).
			Value(&confirmPassphrase).
			Validate(func(s string) error {
				if s != res.Passphrase {
					return errors.New("passphrases do not match")
				}
				return nil
			}),
	).WithHideFunc(func() bool { return keyChoice != keyChoiceCreate })

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
		manualGroup,
		createGroup,
	)

	if err := form.Run(); err != nil {
		return CreateResult{}, err
	}
	res.Name = strings.TrimSpace(res.Name)
	res.GitName = strings.TrimSpace(res.GitName)
	res.GitEmail = strings.TrimSpace(res.GitEmail)

	switch keyChoice {
	case keyChoiceCreate:
		res.NewKey = true
		res.NewKeyPath = strings.TrimSpace(res.NewKeyPath)
		if res.NewKeyPath == "" {
			res.NewKeyPath = defaultKeyPath(res.Name)
		}
		res.SSHKey = res.NewKeyPath
	case keyChoiceManual:
		res.SSHKey = strings.TrimSpace(res.SSHKey)
	default:
		res.SSHKey = keyChoice
	}

	return res, nil
}

// SettingsForm asks whether per-repo automatic masking should be on and
// returns "automatic" or "manual".
func SettingsForm(automatic bool) (string, error) {
	choice := "manual"
	if automatic {
		choice = "automatic"
	}
	err := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Mask control").
				Description("Automatic applies each repo's remembered mask as you cd; manual leaves it to you").
				Options(
					huh.NewOption("Automatic (per-repo)", "automatic"),
					huh.NewOption("Manual", "manual"),
				).
				Value(&choice),
		),
	).Run()
	if err != nil {
		return "", err
	}
	return choice, nil
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

// defaultKeyPath suggests a private key path for a mask name.
func defaultKeyPath(name string) string {
	slug := slugifyKeyName(name)
	if slug == "" {
		slug = "mask"
	}
	return "~/.ssh/id_ed25519_" + slug
}

// slugifyKeyName turns a mask name into a safe file name fragment.
func slugifyKeyName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	var b strings.Builder
	lastDash := false
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '_':
			b.WriteRune(r)
			lastDash = false
		default:
			if b.Len() > 0 && !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

// checkNewKeyPath reports why path cannot be used for a new key.
func checkNewKeyPath(displayPath string) error {
	path := strings.TrimSpace(displayPath)
	if path == "" {
		return errEmpty("SSH key path")
	}
	if strings.HasSuffix(path, ".pub") {
		return errors.New("enter the private key path, without .pub")
	}
	for _, candidate := range []string{path, path + ".pub"} {
		if _, err := os.Stat(config.ExpandPath(candidate)); err == nil {
			return fmt.Errorf("%s already exists", candidate)
		} else if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("cannot check %s: %w", candidate, err)
		}
	}
	return nil
}

// GenerateSSHKey creates an Ed25519 key at path. The passphrase may be empty;
// when set it is fed to ssh-keygen on stdin, so it never appears in the process
// arguments. The comment is typically the git email.
func GenerateSSHKey(path, comment, passphrase string) error {
	if err := checkNewKeyPath(path); err != nil {
		return err
	}
	expanded := config.ExpandPath(strings.TrimSpace(path))
	if err := os.MkdirAll(filepath.Dir(expanded), 0o700); err != nil {
		return fmt.Errorf("create key directory: %w", err)
	}

	args := []string{"-q", "-t", "ed25519", "-f", expanded, "-C", comment}
	if passphrase == "" {
		args = append(args, "-N", "")
	}
	cmd := exec.Command("ssh-keygen", args...)
	if passphrase != "" {
		cmd.Stdin = strings.NewReader(passphrase + "\n" + passphrase + "\n")
	}
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	if err := cmd.Run(); err != nil {
		if msg := strings.TrimSpace(output.String()); msg != "" {
			return fmt.Errorf("ssh-keygen: %w: %s", err, msg)
		}
		return fmt.Errorf("ssh-keygen: %w", err)
	}
	return nil
}

type errEmpty string

func (e errEmpty) Error() string { return string(e) + " is required" }
