package cli

import "github.com/spf13/cobra"

const appName = "mdxs-parser"

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           appName,
		Short:         "mdxs-parser is a cross-platform CLI application",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.AddCommand(newVersionCmd())
	cmd.AddCommand(newCompletionCmd(cmd))

	return cmd
}

func Execute() error {
	return NewRootCmd().Execute()
}
