package cmd

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/soyagvs/relio/internal/ghrelease"
	"github.com/soyagvs/relio/internal/i18n"
)

// authStatus is the shared body of `relio auth status` and the menu's Auth →
// "Show sign-in status" entry. With no token resolvable it must say so without
// touching the network.
func TestAuthStatusSharedBySubcommandAndMenu(t *testing.T) {
	orig := lookupToken
	lookupToken = func() (string, string) { return "", "" }
	t.Cleanup(func() { lookupToken = orig })

	var buf bytes.Buffer
	if err := authStatus(&buf); err != nil {
		t.Fatalf("authStatus: %v", err)
	}
	if !strings.Contains(buf.String(), "not authenticated") {
		t.Errorf("authStatus with no token = %q, want it to mention 'not authenticated'", buf.String())
	}
}

// TestAuthStatusLocalizesOutput proves authStatus' not-authenticated line
// resolves through the active i18n catalog.
func TestAuthStatusLocalizesOutput(t *testing.T) {
	orig := lookupToken
	lookupToken = func() (string, string) { return "", "" }
	t.Cleanup(func() { lookupToken = orig })

	prev := i18n.Current()
	t.Cleanup(func() { i18n.SetLanguage(prev) })

	i18n.SetLanguage("en")
	var bufEN bytes.Buffer
	if err := authStatus(&bufEN); err != nil {
		t.Fatalf("authStatus: %v", err)
	}

	i18n.SetLanguage("es")
	var bufES bytes.Buffer
	if err := authStatus(&bufES); err != nil {
		t.Fatalf("authStatus: %v", err)
	}

	if bufEN.String() == bufES.String() {
		t.Error("authStatus output unchanged across languages")
	}
	if !strings.Contains(bufES.String(), "no autenticado") {
		t.Errorf("es authStatus = %q, want it to mention 'no autenticado'", bufES.String())
	}
}

// TestAuthStatusReportsStoredTokenSource proves authStatus formats an
// arbitrary source string (as returned by tokenstore.Load via
// ghrelease.Token, e.g. "keychain" or "file") the same generic way it already
// handles "gh" or an env var name.
func TestAuthStatusReportsStoredTokenSource(t *testing.T) {
	origLookup := lookupToken
	lookupToken = func() (string, string) { return "tok-abc", "keychain" }
	t.Cleanup(func() { lookupToken = origLookup })

	origAuthUser := authenticatedUser
	authenticatedUser = func(ctx context.Context, client *http.Client, token string) (string, error) {
		return "octocat", nil
	}
	t.Cleanup(func() { authenticatedUser = origAuthUser })

	var buf bytes.Buffer
	if err := authStatus(&buf); err != nil {
		t.Fatalf("authStatus: %v", err)
	}
	if !strings.Contains(buf.String(), "octocat") || !strings.Contains(buf.String(), "keychain") {
		t.Errorf("authStatus = %q, want it to mention octocat and keychain", buf.String())
	}
}

// stubDeviceFlow points every device-flow/tokenstore seam auth.go uses at
// fakes, and restores the originals on test cleanup. It never touches the
// network, a real OS keychain, or a real file.
func stubDeviceFlow(t *testing.T) {
	t.Helper()

	origRequest := requestDeviceCode
	origPoll := pollForToken
	origAuthUser := authenticatedUser
	origStore := storeToken
	origLoad := loadToken
	origDelete := deleteToken

	requestDeviceCode = func(ctx context.Context, client *http.Client, clientID string) (ghrelease.DeviceCode, error) {
		return ghrelease.DeviceCode{
			DeviceCode:      "dc-123",
			UserCode:        "ABCD-1234",
			VerificationURI: "https://github.com/login/device",
			ExpiresIn:       900,
			Interval:        1,
		}, nil
	}
	pollForToken = func(ctx context.Context, client *http.Client, clientID, deviceCode string, interval time.Duration) (string, error) {
		return "tok-xyz", nil
	}
	authenticatedUser = func(ctx context.Context, client *http.Client, token string) (string, error) {
		return "octocat", nil
	}
	storeToken = func(token string) error { return nil }
	loadToken = func() (string, string, error) { return "tok-xyz", "keychain", nil }
	deleteToken = func() error { return nil }

	t.Cleanup(func() {
		requestDeviceCode = origRequest
		pollForToken = origPoll
		authenticatedUser = origAuthUser
		storeToken = origStore
		loadToken = origLoad
		deleteToken = origDelete
	})
}

// TestNewAuthCmdShortLongLocalizeAtConstructionTime proves newAuthCmd()'s
// group Short/Long, the `status` subcommand's Short, and the login/logout
// subcommands' Short and RunE output all resolve through the active catalog
// at construction/run time.
func TestNewAuthCmdShortLongLocalizeAtConstructionTime(t *testing.T) {
	stubDeviceFlow(t)

	prev := i18n.Current()
	t.Cleanup(func() { i18n.SetLanguage(prev) })

	i18n.SetLanguage("en")
	cmdEN := newAuthCmd()
	i18n.SetLanguage("es")
	cmdES := newAuthCmd()

	if cmdES.Short == cmdEN.Short {
		t.Errorf("auth Short unchanged across languages: %q", cmdES.Short)
	}
	if cmdES.Long == cmdEN.Long {
		t.Error("auth Long unchanged across languages")
	}

	findSub := func(c *cobra.Command, name string) *cobra.Command {
		for _, sub := range c.Commands() {
			if sub.Name() == name {
				return sub
			}
		}
		return nil
	}

	statusEN, statusES := findSub(cmdEN, "status"), findSub(cmdES, "status")
	if statusEN == nil || statusES == nil {
		t.Fatal("status subcommand not found")
	}
	if statusES.Short == statusEN.Short {
		t.Errorf("auth status Short unchanged across languages: %q", statusES.Short)
	}

	for _, name := range []string{"login", "logout"} {
		subEN, subES := findSub(cmdEN, name), findSub(cmdES, name)
		if subEN == nil || subES == nil {
			t.Fatalf("%s subcommand not found", name)
		}
		if subES.Short == subEN.Short {
			t.Errorf("auth %s Short unchanged across languages: %q", name, subES.Short)
		}

		// authLogin/authLogout resolve i18n.T() lazily at RunE time (unlike
		// the old static note() bodies captured once at construction), so
		// the active language must be set right before each RunE call.
		i18n.SetLanguage("en")
		var bufEN bytes.Buffer
		subEN.SetOut(&bufEN)
		if err := subEN.RunE(subEN, nil); err != nil {
			t.Fatalf("%s RunE: %v", name, err)
		}

		i18n.SetLanguage("es")
		var bufES bytes.Buffer
		subES.SetOut(&bufES)
		if err := subES.RunE(subES, nil); err != nil {
			t.Fatalf("%s RunE: %v", name, err)
		}
		if bufEN.String() == bufES.String() {
			t.Errorf("auth %s body unchanged across languages: %q", name, bufEN.String())
		}
	}
}

// TestAuthLoginSuccess proves a full, successful device-flow login: it shows
// the verification URL and user code, waits, then stores and confirms the
// resulting token.
func TestAuthLoginSuccess(t *testing.T) {
	stubDeviceFlow(t)

	var storedWith string
	storeToken = func(token string) error {
		storedWith = token
		return nil
	}

	var buf bytes.Buffer
	if err := authLogin(&buf); err != nil {
		t.Fatalf("authLogin: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "https://github.com/login/device") {
		t.Errorf("authLogin output = %q, want it to contain the verification URL", out)
	}
	if !strings.Contains(out, "ABCD-1234") {
		t.Errorf("authLogin output = %q, want it to contain the user code", out)
	}
	if !strings.Contains(out, "octocat") {
		t.Errorf("authLogin output = %q, want it to contain the logged-in username", out)
	}
	if storedWith != "tok-xyz" {
		t.Errorf("storeToken called with %q, want the polled access token", storedWith)
	}
}

// TestAuthLoginExpiredCode proves an expired device code is reported as a
// plain, non-error outcome telling the user to try again.
func TestAuthLoginExpiredCode(t *testing.T) {
	stubDeviceFlow(t)

	pollForToken = func(ctx context.Context, client *http.Client, clientID, deviceCode string, interval time.Duration) (string, error) {
		return "", ghrelease.ErrExpiredToken
	}
	var storeCalled bool
	storeToken = func(token string) error {
		storeCalled = true
		return nil
	}

	var buf bytes.Buffer
	if err := authLogin(&buf); err != nil {
		t.Fatalf("authLogin: %v, want nil (expiry is not a hard error)", err)
	}
	if storeCalled {
		t.Error("storeToken must not be called after an expired code")
	}
	if out := buf.String(); !strings.Contains(strings.ToLower(out), "expired") {
		t.Errorf("authLogin output = %q, want it to mention the code expired", out)
	}
}

// TestAuthLoginAccessDenied proves a declined authorization is reported as a
// plain, non-error outcome.
func TestAuthLoginAccessDenied(t *testing.T) {
	stubDeviceFlow(t)

	pollForToken = func(ctx context.Context, client *http.Client, clientID, deviceCode string, interval time.Duration) (string, error) {
		return "", ghrelease.ErrAccessDenied
	}
	var storeCalled bool
	storeToken = func(token string) error {
		storeCalled = true
		return nil
	}

	var buf bytes.Buffer
	if err := authLogin(&buf); err != nil {
		t.Fatalf("authLogin: %v, want nil (a decline is not a hard error)", err)
	}
	if storeCalled {
		t.Error("storeToken must not be called after a declined authorization")
	}
	if out := buf.String(); !strings.Contains(strings.ToLower(out), "declined") {
		t.Errorf("authLogin output = %q, want it to mention the declined authorization", out)
	}
}

// TestAuthLoginPollOtherErrorPropagates proves an unexpected poll failure (a
// real HTTP/network error, not one of the sentinel outcomes above) surfaces
// as a real command error rather than being swallowed.
func TestAuthLoginPollOtherErrorPropagates(t *testing.T) {
	stubDeviceFlow(t)

	wantErr := errors.New("network boom")
	pollForToken = func(ctx context.Context, client *http.Client, clientID, deviceCode string, interval time.Duration) (string, error) {
		return "", wantErr
	}

	var buf bytes.Buffer
	if err := authLogin(&buf); !errors.Is(err, wantErr) {
		t.Fatalf("authLogin err = %v, want %v", err, wantErr)
	}
}

// TestAuthLoginRequestDeviceCodeErrorPropagates proves a failure requesting
// the device code itself is a real command error.
func TestAuthLoginRequestDeviceCodeErrorPropagates(t *testing.T) {
	stubDeviceFlow(t)

	wantErr := errors.New("device code request boom")
	requestDeviceCode = func(ctx context.Context, client *http.Client, clientID string) (ghrelease.DeviceCode, error) {
		return ghrelease.DeviceCode{}, wantErr
	}

	var buf bytes.Buffer
	if err := authLogin(&buf); !errors.Is(err, wantErr) {
		t.Fatalf("authLogin err = %v, want %v", err, wantErr)
	}
}

// TestAuthLogoutSuccess proves logging out with a stored token deletes it and
// confirms it.
func TestAuthLogoutSuccess(t *testing.T) {
	stubDeviceFlow(t)

	var deleteCalled bool
	deleteToken = func() error {
		deleteCalled = true
		return nil
	}

	var buf bytes.Buffer
	if err := authLogout(&buf); err != nil {
		t.Fatalf("authLogout: %v", err)
	}
	if !deleteCalled {
		t.Error("deleteToken was not called")
	}
	if out := buf.String(); !strings.Contains(strings.ToLower(out), "logged out") {
		t.Errorf("authLogout output = %q, want it to confirm logout", out)
	}
}

// TestAuthLogoutNothingStored proves logging out with nothing stored still
// succeeds, per tokenstore.Delete's contract, but reports a distinct message.
func TestAuthLogoutNothingStored(t *testing.T) {
	stubDeviceFlow(t)

	loadToken = func() (string, string, error) { return "", "", nil }
	var deleteCalled bool
	deleteToken = func() error {
		deleteCalled = true
		return nil
	}

	var buf bytes.Buffer
	if err := authLogout(&buf); err != nil {
		t.Fatalf("authLogout: %v, want nil (nothing stored is not an error)", err)
	}
	if !deleteCalled {
		t.Error("deleteToken was not called")
	}
	if out := buf.String(); !strings.Contains(strings.ToLower(out), "nothing") {
		t.Errorf("authLogout output = %q, want it to mention nothing was stored", out)
	}
}

// TestAuthLogoutDeleteErrorPropagates proves a genuine deleteToken failure is
// a real command error.
func TestAuthLogoutDeleteErrorPropagates(t *testing.T) {
	stubDeviceFlow(t)

	wantErr := errors.New("delete boom")
	deleteToken = func() error { return wantErr }

	var buf bytes.Buffer
	if err := authLogout(&buf); !errors.Is(err, wantErr) {
		t.Fatalf("authLogout err = %v, want %v", err, wantErr)
	}
}

// TestAuthLogoutLoadErrorPropagates proves a genuine loadToken failure (a
// real file-permission problem, per tokenstore.Load's contract) is a real
// command error rather than being treated as "nothing stored".
func TestAuthLogoutLoadErrorPropagates(t *testing.T) {
	stubDeviceFlow(t)

	wantErr := errors.New("load boom")
	loadToken = func() (string, string, error) { return "", "", wantErr }

	var buf bytes.Buffer
	if err := authLogout(&buf); !errors.Is(err, wantErr) {
		t.Fatalf("authLogout err = %v, want %v", err, wantErr)
	}
}
