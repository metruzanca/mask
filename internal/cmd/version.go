package cmd

import (
	"fmt"
	"io"
	"runtime/debug"

	"github.com/spf13/cobra"
)

// Version and Commit are set at release time by GoReleaser with
// -ldflags "-X github.com/metruzanca/mask/internal/cmd.Version={{.Tag}} ...".
// Locally they stay empty, so buildVersion falls back to the module version
// that `go install` records and to the local git revision.
var (
	Version = ""
	Commit  = ""
)

const devVersion = "dev"

func newVersion() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the mask version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			version, commit := buildVersion()
			_, err := io.WriteString(cmd.OutOrStdout(), formatVersion(version, commit)+"\n")
			return err
		},
	}
}

// buildVersion prefers the release-time ldflags, then the version embedded by
// `go install module@version`, and finally the local git revision.
func buildVersion() (string, string) {
	version, commit := Version, Commit
	info, ok := debug.ReadBuildInfo()
	if !ok {
		if version == "" {
			version = devVersion
		}
		return version, commit
	}
	if version == "" {
		if v := info.Main.Version; v != "" && v != "(devel)" {
			version = v
		} else {
			version = devVersion
		}
	}
	if commit == "" {
		for _, s := range info.Settings {
			if s.Key == "vcs.revision" {
				commit = s.Value
				break
			}
		}
	}
	return version, commit
}

// formatVersion renders "mask v0.3.0 (26d7c6d)", omitting the commit when it
// is unknown. It takes the values as arguments so tests do not depend on how
// the binary was built.
func formatVersion(version, commit string) string {
	if version == "" {
		version = devVersion
	}
	line := "mask " + version
	if len(commit) > 7 {
		commit = commit[:7]
	}
	if commit != "" {
		line += fmt.Sprintf(" (%s)", commit)
	}
	return line
}
