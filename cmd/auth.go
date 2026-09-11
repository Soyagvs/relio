package cmd

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/spf13/cobra"

	"github.com/soyagvs/relio/internal/ghrelease"
	"github.com/soyagvs/relio/internal/i18n"
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
		fmt.Fprintln(w, ui.Info(i18n.T(i18n.AuthNotAuthenticated)))
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	login, err := ghrelease.AuthenticatedUser(ctx, nil, token)
	if err != nil {
		return err
	}
	fmt.Fprintln(w, ui.Info(i18n.T(i18n.AuthLoggedInAs, login, source)))
	return nil
}

// newAuthCmd groups the GitHub token helpers. Relio has no login of its own yet:
// it reads a personal access token from RELIO_GITHUB_TOKEN / GITHUB_TOKEN /
// GH_TOKEN, or from `gh auth token` when the GitHub CLI is signed in.
func newAuthCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "auth",
		Short: i18n.T(i18n.AuthShort),
		Long:  i18n.T(i18n.AuthLong),
	}

	status := &cobra.Command{
		Use:   "status",
		Short: i18n.T(i18n.AuthStatusShort),
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
		note("login", i18n.T(i18n.AuthLoginShort), i18n.T(i18n.AuthLoginBody)),
		status,
		note("logout", i18n.T(i18n.AuthLogoutShort), i18n.T(i18n.AuthLogoutBody)),
	)
	return c
}
