package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"

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

// TestNewAuthCmdShortLongLocalizeAtConstructionTime proves newAuthCmd()'s
// group Short/Long, the `status` subcommand's Short, and the login/logout
// note commands' Short/body all resolve through the active catalog at
// construction time.
func TestNewAuthCmdShortLongLocalizeAtConstructionTime(t *testing.T) {
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

		var bufEN, bufES bytes.Buffer
		subEN.SetOut(&bufEN)
		subES.SetOut(&bufES)
		if err := subEN.RunE(subEN, nil); err != nil {
			t.Fatalf("%s RunE: %v", name, err)
		}
		if err := subES.RunE(subES, nil); err != nil {
			t.Fatalf("%s RunE: %v", name, err)
		}
		if bufEN.String() == bufES.String() {
			t.Errorf("auth %s body unchanged across languages: %q", name, bufEN.String())
		}
	}
}

// TestGoldenEnglishAuthCmdUnchanged pins the hardcoded English literals in
// cmd/auth.go against i18n.T() under the default "en" language: converting
// them to i18n.T() calls MUST NOT change a single byte of English output.
func TestGoldenEnglishAuthCmdUnchanged(t *testing.T) {
	prev := i18n.Current()
	if prev != "en" {
		i18n.SetLanguage("en")
	}
	t.Cleanup(func() { i18n.SetLanguage(prev) })

	c := newAuthCmd()
	if c.Short != "Inspect the GitHub token relio will use" {
		t.Errorf("Short = %q", c.Short)
	}
	wantLong := "Relio authenticates to GitHub with a personal access token, not its own login.\n" +
		"It checks RELIO_GITHUB_TOKEN, GITHUB_TOKEN and GH_TOKEN in that order, then\n" +
		"falls back to `gh auth token`. `status` shows which one was found and who it\n" +
		"belongs to."
	if c.Long != wantLong {
		t.Errorf("Long = %q, want %q", c.Long, wantLong)
	}

	var login, logout, status *cobra.Command
	for _, sub := range c.Commands() {
		switch sub.Name() {
		case "login":
			login = sub
		case "logout":
			logout = sub
		case "status":
			status = sub
		}
	}
	if status == nil || status.Short != "Show which token relio found and who it belongs to" {
		t.Errorf("status Short = %q", status.Short)
	}
	if login == nil || login.Short != "How to give relio a GitHub token" {
		t.Errorf("login Short = %q", login.Short)
	}
	if logout == nil || logout.Short != "How to drop the GitHub token" {
		t.Errorf("logout Short = %q", logout.Short)
	}

	var buf bytes.Buffer
	login.SetOut(&buf)
	if err := login.RunE(login, nil); err != nil {
		t.Fatalf("login RunE: %v", err)
	}
	if want := "No device-flow login yet. Set GITHUB_TOKEN to a PAT with `repo` scope, or run `gh auth login` and relio will reuse the `gh` token."; !strings.Contains(buf.String(), want) {
		t.Errorf("login output = %q, want it to contain %q", buf.String(), want)
	}

	buf.Reset()
	logout.SetOut(&buf)
	if err := logout.RunE(logout, nil); err != nil {
		t.Fatalf("logout RunE: %v", err)
	}
	if want := "Relio stores nothing. Unset RELIO_GITHUB_TOKEN / GITHUB_TOKEN / GH_TOKEN, or run `gh auth logout`."; !strings.Contains(buf.String(), want) {
		t.Errorf("logout output = %q, want it to contain %q", buf.String(), want)
	}
}
