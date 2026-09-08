package cmd

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/spf13/cobra"

	"github.com/soyagvs/relio/internal/ghrelease"
	"github.com/soyagvs/relio/internal/ui"
)

// lookupToken resolves the GitHub token relio will use. It is a package var so
// tests can stub token resolution without touching the environment or `gh`.
var lookupToken = ghrelease.Token

// authStatus reports which GitHub token relio found and who it belongs to. It is
// the shared body of `relio auth status` and the menu's Auth entry.
func authStatus(w io.Writer) error {
	token, source := lookupToken()
	if token == "" {
		fmt.Fprintln(w, ui.Info("not authenticated (set GITHUB_TOKEN or run `gh auth login`)"))
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	login, err := ghrelease.AuthenticatedUser(ctx, nil, token)
	if err != nil {
		return err
	}
	fmt.Fprintln(w, ui.Info(fmt.Sprintf("logged in as %s (via %s)", login, source)))
	return nil
}

// newAuthCmd groups the GitHub token helpers. Relio has no login of its own yet:
// it reads a personal access token from RELIO_GITHUB_TOKEN / GITHUB_TOKEN /
// GH_TOKEN, or from `gh auth token` when the GitHub CLI is signed in.
func newAuthCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "auth",
		Short: "Inspect the GitHub token relio will use",
		Long: "Relio authenticates to GitHub with a personal access token, not its own login.\n" +
			"It checks RELIO_GITHUB_TOKEN, GITHUB_TOKEN and GH_TOKEN in that order, then\n" +
			"falls back to `gh auth token`. `status` shows which one was found and who it\n" +
			"belongs to.",
	}

	status := &cobra.Command{
		Use:   "status",
		Short: "Show which token relio found and who it belongs to",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return authStatus(cmd.OutOrStdout())
		},
	}

	note := func(use, short, body string) *cobra.Command {
		return &cobra.Command{
			Use:   use,
			Short: short,
			Args:  cobra.NoArgs,
			RunE: func(cmd *cobra.Command, args []string) error {
				fmt.Fprintln(cmd.OutOrStdout(), ui.Info(body))
				return nil
			},
		}
	}

	c.AddCommand(
		note("login", "How to give relio a GitHub token",
			"No device-flow login yet. Set GITHUB_TOKEN to a PAT with `repo` scope, or run "+
				"`gh auth login` and relio will reuse the `gh` token."),
		status,
		note("logout", "How to drop the GitHub token",
			"Relio stores nothing. Unset RELIO_GITHUB_TOKEN / GITHUB_TOKEN / GH_TOKEN, or run "+
				"`gh auth logout`."),
	)
	return c
}
