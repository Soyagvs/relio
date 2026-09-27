package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/spf13/cobra"

	"github.com/soyagvs/relio/internal/ghrelease"
	"github.com/soyagvs/relio/internal/i18n"
	"github.com/soyagvs/relio/internal/tokenstore"
	"github.com/soyagvs/relio/internal/ui"
)

// lookupToken resolves the GitHub token relio will use. It is a package var so
// tests can stub token resolution without touching the environment or `gh`.
var lookupToken = ghrelease.Token

// requestDeviceCode, pollForToken and authenticatedUser are package vars
// aliasing their internal/ghrelease counterparts so `relio auth login`'s
// tests can inject fakes without hitting real GitHub HTTP endpoints.
var (
	requestDeviceCode = ghrelease.RequestDeviceCode
	pollForToken      = ghrelease.PollForToken
	authenticatedUser = ghrelease.AuthenticatedUser
)

// storeToken, loadToken and deleteToken are package vars aliasing their
// internal/tokenstore counterparts so `relio auth login`/`logout`'s tests
// never touch a real OS keychain or file.
var (
	storeToken  = tokenstore.Store
	loadToken   = tokenstore.Load
	deleteToken = tokenstore.Delete
)

// authStatus reports which GitHub token relio found and who it belongs to. It is
// the shared body of `relio auth status` and the menu's Auth entry.
func authStatus(w io.Writer) error {
	token, source := lookupToken()
	if token == "" {
		_, err := fmt.Fprintln(w, ui.Info(i18n.T(i18n.AuthNotAuthenticated)))
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	login, err := authenticatedUser(ctx, nil, token)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w, ui.Info(i18n.T(i18n.AuthLoggedInAs, login, source)))
	return err
}

// authLogin runs `relio auth login`'s OAuth Device Flow: it asks GitHub for a
// device code, shows the user where to go and what to enter, polls until the
// user finishes (or the code expires, or they decline), then stores the
// resulting token and confirms who it belongs to.
//
// An expired code or a declined authorization are reported to w as ordinary
// (non-error) outcomes -- the user can simply run `relio auth login` again --
// mirroring authStatus's "not authenticated" convention. Any other failure
// (the device code request, the poll's underlying HTTP calls, storing the
// token, or confirming the authenticated user) is returned as a real error.
func authLogin(w io.Writer) error {
	ctx := context.Background()

	dc, err := requestDeviceCode(ctx, nil, ghrelease.RelioOAuthClientID)
	if err != nil {
		return err
	}

	if _, err := fmt.Fprintln(w, ui.Info(i18n.T(i18n.AuthLoginInstruction, dc.VerificationURI, dc.UserCode))); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, ui.Info(i18n.T(i18n.AuthLoginWaiting))); err != nil {
		return err
	}

	pollCtx, cancel := context.WithTimeout(ctx, time.Duration(dc.ExpiresIn)*time.Second)
	defer cancel()

	token, err := pollForToken(pollCtx, nil, ghrelease.RelioOAuthClientID, dc.DeviceCode, time.Duration(dc.Interval)*time.Second)
	if err != nil {
		switch {
		case errors.Is(err, ghrelease.ErrExpiredToken), errors.Is(err, context.DeadlineExceeded):
			_, werr := fmt.Fprintln(w, ui.Info(i18n.T(i18n.AuthLoginExpired)))
			return werr
		case errors.Is(err, ghrelease.ErrAccessDenied):
			_, werr := fmt.Fprintln(w, ui.Info(i18n.T(i18n.AuthLoginDenied)))
			return werr
		default:
			return err
		}
	}

	if err := storeToken(token); err != nil {
		return err
	}

	login, err := authenticatedUser(ctx, nil, token)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(w, ui.Info(i18n.T(i18n.AuthLoginSuccess, login)))
	return err
}

// authLogout runs `relio auth logout`: it removes whatever GitHub token
// `relio auth login` stored. Nothing having been stored is not an error (it
// mirrors tokenstore.Delete's own contract) -- authLogout reports it as a
// distinct, still-successful outcome instead.
func authLogout(w io.Writer) error {
	tok, _, err := loadToken()
	if err != nil {
		return err
	}

	if err := deleteToken(); err != nil {
		return err
	}

	if tok == "" {
		_, err := fmt.Fprintln(w, ui.Info(i18n.T(i18n.AuthLogoutNothingStored)))
		return err
	}
	_, err = fmt.Fprintln(w, ui.Info(i18n.T(i18n.AuthLogoutSuccess)))
	return err
}

// newAuthCmd groups the GitHub token helpers: `relio auth login`/`logout` run
// the OAuth Device Flow and store the resulting token; `status` shows which
// token relio found and who it belongs to.
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

	login := &cobra.Command{
		Use:   "login",
		Short: i18n.T(i18n.AuthLoginShort),
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return authLogin(cmd.OutOrStdout())
		},
	}

	logout := &cobra.Command{
		Use:   "logout",
		Short: i18n.T(i18n.AuthLogoutShort),
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return authLogout(cmd.OutOrStdout())
		},
	}

	c.AddCommand(login, status, logout)
	return c
}
