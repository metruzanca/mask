package cmd

import (
	"strings"
	"testing"
)

func TestFormatVersion(t *testing.T) {
	cases := []struct {
		version string
		commit  string
		want    string
	}{
		{"v0.3.0", "26d7c6d7ce78998eda99a4caa3bb7cec175fe98b", "mask v0.3.0 (26d7c6d)"},
		{"v0.3.0", "", "mask v0.3.0"},
		{"", "abc", "mask dev (abc)"},
		{"", "", "mask dev"},
	}
	for _, c := range cases {
		if got := formatVersion(c.version, c.commit); got != c.want {
			t.Errorf("formatVersion(%q, %q) = %q, want %q", c.version, c.commit, got, c.want)
		}
	}
}

func TestVersionCommandUsesReleaseVariables(t *testing.T) {
	oldVersion, oldCommit := Version, Commit
	Version, Commit = "v1.2.3", "abcdef1234567890"
	t.Cleanup(func() { Version, Commit = oldVersion, oldCommit })

	stdout, _, err := runMask(t, "version")
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(stdout); got != "mask v1.2.3 (abcdef1)" {
		t.Errorf("stdout = %q, want release version", got)
	}
}
