package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/soyagvs/relio/internal/ui"
)

// newAuthCmd is a placeholder for the v0.2.0 GitHub integration. It is wired in
// now so `release --help` documents the planned surface, but the subcommands
// intentionally do nothing yet.
func newAuthCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "auth",
		Short: "Manage GitHub authentication (coming in v0.2.0)",
		Long: "Per-user GitHub login via OAuth Device Flow, with credentials stored in the OS keychain.\n" +
			"Not implemented yet — this command is a preview of the v0.2.0 surface.",
	}

	notYet := func(use, short string) *cobra.Command {
		return &cobra.Command{
			Use:   use,
			Short: short,
			Args:  cobra.NoArgs,
			RunE: func(cmd *cobra.Command, args []string) error {
				fmt.Fprintln(cmd.OutOrStdout(), ui.Info("`relio auth "+use+"` lands in v0.2.0."))
				return nil
			},
		}
	}

	c.AddCommand(
		notYet("login", "Authenticate with your own GitHub account"),
		notYet("status", "Show who you are logged in as"),
		notYet("logout", "Remove local credentials"),
	)
	return c
}
