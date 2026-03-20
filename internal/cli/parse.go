package cli

import (
	"fmt"

	"github.com/owner/mdxs-parser/internal/parser"
	"github.com/spf13/cobra"
)

func newParseCmd() *cobra.Command {
	var (
		jsonOutput     bool
		markdownOutput bool
	)

	cmd := &cobra.Command{
		Use:   "parse <file>",
		Short: "Parse markdown into JSON or expanded markdown",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if jsonOutput && markdownOutput {
				return fmt.Errorf("--json and --markdown cannot be used together")
			}

			path := args[0]
			if markdownOutput {
				output, err := parser.RenderMarkdown(path)
				if err != nil {
					return err
				}
				_, err = fmt.Fprintln(cmd.OutOrStdout(), output)
				return err
			}

			output, err := parser.RenderJSON(path)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), string(output))
			return err
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output JSON")
	cmd.Flags().BoolVar(&markdownOutput, "markdown", false, "Output expanded markdown")

	return cmd
}
