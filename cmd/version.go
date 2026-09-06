package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/soyagvs/relio/internal/ui"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the Relio version",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintf(cmd.OutOrStdout(), "%s %s (commit %s, built %s)\n", ui.AppName, version, commit, date)
		},
	}
}
