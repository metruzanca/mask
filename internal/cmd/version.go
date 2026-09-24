package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

// Version is the build version. It is overridden at release time with
// -ldflags "-X github.com/metruzanca/mask/internal/cmd.Version=vX.Y.Z".
var Version = "dev"

func newVersion() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the mask version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := io.WriteString(cmd.OutOrStdout(), fmt.Sprintf("mask %s\n", Version))
			return err
		},
	}
}
