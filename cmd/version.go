package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/soyagvs/relio/internal/ui"
	"github.com/soyagvs/relio/internal/update"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the Relio version",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			w := cmd.OutOrStdout()
			fmt.Fprintf(w, "%s %s (commit %s, built %s)\n", ui.AppName, version, commit, date)

			// Only nudge on a real terminal, so scripts parsing `relio version`
			// keep getting a single clean line.
			if stdoutIsTTY() {
				if v := update.Available(version); v != "" {
					fmt.Fprintln(w, ui.Key.Render("▲ v"+v+" available — brew upgrade relio"))
				}
			}
		},
	}
}
