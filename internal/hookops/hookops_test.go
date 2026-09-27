package hookops

import (
	"testing"

	"github.com/metruzanca/mask/internal/repo"
)

func base() Input {
	return Input{
		Automatic:   true,
		MaskExists:  func(string) bool { return true },
		InRepo:      true,
		Repo:        repo.Info{Root: "/r", GitDir: "/r/.git", ID: "id"},
		CurrentRepo: "old",
	}
}

func TestDisabledDoesNothing(t *testing.T) {
	in := base()
	in.Automatic = false
	if d := Decide(in); d.Action != ActionNone {
		t.Errorf("action = %v", d.Action)
	}
}

func TestSameRepoDoesNothing(t *testing.T) {
	in := base()
	in.CurrentRepo = "id"
	in.CurrentMask = ""
	in.Cache = CacheMask
	in.CachedMask = "work"
	if d := Decide(in); d.Action != ActionNone {
		t.Errorf("action = %v", d.Action)
	}
}

func TestEnterCachedSwitches(t *testing.T) {
	in := base()
	in.Cache = CacheMask
	in.CachedMask = "work"
	in.CurrentMask = "personal"
	d := Decide(in)
	if d.Action != ActionSwitch || d.Mask != "work" {
		t.Fatalf("got %+v", d)
	}
	if d.RepoID != "id" || d.Notice == "" || d.Log == "" {
		t.Errorf("missing repo/notice/log: %+v", d)
	}
}

func TestEnterCachedAlreadyWorn(t *testing.T) {
	in := base()
	in.Cache = CacheMask
	in.CachedMask = "work"
	in.CurrentMask = "work"
	if d := Decide(in); d.Action != ActionNone {
		t.Errorf("action = %v", d.Action)
	}
}

func TestEnterCachedNoneTurnsOff(t *testing.T) {
	in := base()
	in.Cache = CacheNone
	in.CurrentMask = "personal"
	d := Decide(in)
	if d.Action != ActionOff || d.RepoID != "id" {
		t.Fatalf("got %+v", d)
	}
}

func TestEnterCachedNoneAlreadyOff(t *testing.T) {
	in := base()
	in.Cache = CacheNone
	in.CurrentMask = ""
	if d := Decide(in); d.Action != ActionNone || d.RepoID != "id" {
		t.Errorf("got %+v", d)
	}
}

func TestEnterUncachedTurnsOff(t *testing.T) {
	in := base()
	in.CurrentMask = "personal"
	d := Decide(in)
	if d.Action != ActionOff || d.RepoID != "id" {
		t.Fatalf("got %+v", d)
	}
}

func TestEnterUncachedWithNoMaskDoesNothing(t *testing.T) {
	in := base()
	in.CurrentMask = ""
	if d := Decide(in); d.Action != ActionNone {
		t.Errorf("action = %v", d.Action)
	}
}

func TestEnterStaleCacheForgets(t *testing.T) {
	in := base()
	in.Cache = CacheMask
	in.CachedMask = "gone"
	in.MaskExists = func(string) bool { return false }
	in.CurrentMask = "personal"
	d := Decide(in)
	if d.Action != ActionOff || !d.Forget {
		t.Fatalf("got %+v", d)
	}
}

func TestEnterStaleCacheWithNoMaskForgetsOnly(t *testing.T) {
	in := base()
	in.Cache = CacheMask
	in.CachedMask = "gone"
	in.MaskExists = func(string) bool { return false }
	in.CurrentMask = ""
	d := Decide(in)
	if d.Action != ActionNone || !d.Forget {
		t.Fatalf("got %+v", d)
	}
}

func TestLeaveRepoTurnsOff(t *testing.T) {
	in := base()
	in.InRepo = false
	in.Repo = repo.Info{}
	in.CurrentMask = "work"
	d := Decide(in)
	if d.Action != ActionOff || d.RepoID != "" {
		t.Fatalf("got %+v", d)
	}
}

func TestLeaveRepoNoMaskDoesNothing(t *testing.T) {
	in := base()
	in.InRepo = false
	in.Repo = repo.Info{}
	in.CurrentMask = ""
	in.CurrentRepo = "id"
	if d := Decide(in); d.Action != ActionNone || d.RepoID != "" {
		t.Errorf("got %+v", d)
	}
}
