// Package hookops decides what the automatic directory hook should do when the
// shell changes directory. It is UI- and IO-free so the whole precedence matrix
// can be unit-tested: entering a repository applies its remembered mask, leaving
// one takes the mask off, and nested repositories resolve innermost-first.
package hookops

import (
	"fmt"

	"github.com/metruzanca/mask/internal/repo"
)

// Action is the environment change the hook should apply.
type Action int

const (
	// ActionNone leaves the current mask untouched.
	ActionNone Action = iota
	// ActionOff removes the current mask.
	ActionOff
	// ActionSwitch applies Mask.
	ActionSwitch
)

// Input is everything Decide needs to know.
type Input struct {
	// Automatic is the [settings] automatic toggle.
	Automatic bool
	// CurrentMask is $MASK_NAME in the calling shell.
	CurrentMask string
	// CurrentRepo is $MASK_REPO in the calling shell.
	CurrentRepo string
	// InRepo reports whether the shell is inside a repository.
	InRepo bool
	// Repo is the resolved repository when InRepo is true.
	Repo repo.Info
	// CachedMask and HasCache describe the remembered mask for Repo.
	CachedMask string
	HasCache   bool
	// MaskExists reports whether a mask name is still configured.
	MaskExists func(string) bool
}

// Decision is the outcome. RepoID is always the repo id the hook should carry
// in MASK_REPO (empty to unset), so the next transition can be detected even
// when the mask itself does not change.
type Decision struct {
	Action Action
	Mask   string
	RepoID string
	// Notice is shown to the user when a change happens.
	Notice string
	// Log is the line appended to the automatic-change log.
	Log string
	// Forget is true when the cached entry points at a mask that no longer
	// exists and should be dropped.
	Forget bool
}

// Decide computes the hook action.
func Decide(in Input) Decision {
	nowID := ""
	if in.InRepo {
		nowID = in.Repo.ID
	}
	d := Decision{RepoID: nowID}

	if !in.Automatic {
		return d
	}
	if nowID == in.CurrentRepo {
		return d
	}

	if !in.InRepo {
		if in.CurrentMask == "" {
			return d
		}
		d.Action = ActionOff
		d.Notice = "left repo -> mask off"
		d.Log = d.Notice
		return d
	}

	root := in.Repo.Root
	switch {
	case in.HasCache && in.MaskExists != nil && in.MaskExists(in.CachedMask):
		if in.CachedMask == in.CurrentMask {
			return d
		}
		d.Action = ActionSwitch
		d.Mask = in.CachedMask
		d.Notice = fmt.Sprintf("entered %s -> wearing %q", root, in.CachedMask)
		d.Log = d.Notice
	case in.HasCache:
		// Remembered mask was deleted from the config.
		d.Forget = true
		if in.CurrentMask == "" {
			return d
		}
		d.Action = ActionOff
		d.Notice = fmt.Sprintf("entered %s -> remembered mask %q gone, mask off", root, in.CachedMask)
		d.Log = d.Notice
	default:
		if in.CurrentMask == "" {
			return d
		}
		d.Action = ActionOff
		d.Notice = fmt.Sprintf("entered %s -> no mask remembered, mask off", root)
		d.Log = d.Notice
	}
	return d
}
