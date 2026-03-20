package cli

import (
	"fmt"

	"github.com/owner/mdxs-parser/internal/version"
	"github.com/spf13/cobra"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "%s version %s\ncommit: %s\nbuilt: %s\n", appName, version.Version, version.Commit, version.Date)
			return err
		},
	}
}
