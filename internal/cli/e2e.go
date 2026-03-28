package cli

import (
	"context"
	"fmt"

	"github.com/owner/mdxs-parser/internal/e2e"
	"github.com/spf13/cobra"
)

func newE2ECmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "e2e [spec-dir]",
		Short: "Run static E2E tests from markdown specs",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			specDir := "e2e-mdxs"
			if len(args) == 1 {
				specDir = args[0]
			}

			if err := e2e.Run(context.Background(), specDir, cmd.OutOrStdout()); err != nil {
				return fmt.Errorf("run e2e specs: %w", err)
			}
			return nil
		},
	}

	return cmd
}
