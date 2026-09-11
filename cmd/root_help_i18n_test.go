package cmd

import (
	"strings"
	"testing"

	"github.com/soyagvs/relio/internal/i18n"
)

// lookupFlag finds name in either the command's local or persistent flag set.
func lookupFlag(t *testing.T, name string) *flagUsage {
	t.Helper()
	c := NewRootCmd()
	f := c.Flags().Lookup(name)
	if f == nil {
		f = c.PersistentFlags().Lookup(name)
	}
	if f == nil {
		t.Fatalf("flag %q not found on root command", name)
	}
	return &flagUsage{name: name, usage: f.Usage}
}

type flagUsage struct {
	name  string
	usage string
}

// TestHelpAndRootLocalizeAtConstructionTime proves that Execute()'s ordering
// — resolving the active language BEFORE NewRootCmd() builds the command
// tree — actually matters: cobra bakes Short/Long/flag-usage text into the
// struct at construction time, so switching the active language and
// re-constructing the tree must change that text. It also proves
// cmd/help.go's helpReference() and cmd/version.go's update-available
// template resolve through the active catalog.
func TestHelpAndRootLocalizeAtConstructionTime(t *testing.T) {
	prev := i18n.Current()
	t.Cleanup(func() { i18n.SetLanguage(prev) })

	i18n.SetLanguage("en")
	rootEN := NewRootCmd()
	yesEN := lookupFlag(t, "yes")
	i18n.SetLanguage("en")

	i18n.SetLanguage("es")
	rootES := NewRootCmd()
	yesES := lookupFlag(t, "yes")

	if rootES.Short == rootEN.Short {
		t.Errorf("root.Short unchanged across languages: %q", rootES.Short)
	}
	if rootES.Long == rootEN.Long {
		t.Error("root.Long unchanged across languages")
	}
	if yesES.usage == yesEN.usage {
		t.Errorf("--yes usage unchanged across languages: %q", yesES.usage)
	}

	i18n.SetLanguage("en")
	helpEN := helpReference()
	i18n.SetLanguage("es")
	helpES := helpReference()
	if helpEN == "" || helpES == "" {
		t.Fatal("helpReference() returned an empty string")
	}
	if helpEN == helpES {
		t.Error("helpReference() unchanged across languages")
	}

	i18n.SetLanguage("en")
	upEN := i18n.T(i18n.VersionUpdateAvailable, "9.9.9")
	i18n.SetLanguage("es")
	upES := i18n.T(i18n.VersionUpdateAvailable, "9.9.9")
	if upEN == upES {
		t.Errorf("version update-available text unchanged across languages: %q", upEN)
	}
	if !strings.Contains(upEN, "9.9.9") || !strings.Contains(upES, "9.9.9") {
		t.Errorf("version update-available text lost the version arg: en=%q es=%q", upEN, upES)
	}
}

// TestGoldenEnglishHelpUnchanged pins every hardcoded English literal in
// cmd/root.go's Short/Long/flag usage, cmd/help.go's helpReference(), and
// cmd/version.go's update-available template against i18n.T() under the
// default "en" language: converting those literals to i18n.T() calls MUST
// NOT change a single byte of English output.
func TestGoldenEnglishHelpUnchanged(t *testing.T) {
	prev := i18n.Current()
	if prev != "en" {
		i18n.SetLanguage("en")
	}
	t.Cleanup(func() { i18n.SetLanguage(prev) })

	root := NewRootCmd()
	if root.Short != "Turn finished code into a published release" {
		t.Errorf("root.Short = %q", root.Short)
	}
	wantLongSuffix := "  Read the repo's git activity and turn it into a version, changelog,\n" +
		"  and tag — in one command, with a preview before anything is written.\n\n" +
		"  Run `relio` on its own for the interactive menu. Use the\n" +
		"  subcommands below for setup, extras, and scripting."
	if !strings.HasSuffix(root.Long, wantLongSuffix) {
		t.Errorf("root.Long = %q, want suffix %q", root.Long, wantLongSuffix)
	}

	wantFlagUsage := map[string]string{
		"dir":              "run as if relio was started in `path`",
		"no-hash":          "hide commit hashes in release notes",
		"patch":            "force a PATCH bump",
		"minor":            "force a MINOR bump",
		"major":            "force a MAJOR bump",
		"yes":              "skip the interactive menu and confirmation",
		"no-changelog":     "do not touch the changelog file",
		"no-tag":           "do not create the git tag",
		"no-version-files": "do not update the files listed in version_files",
		"publish":          "push and create the GitHub Release after tagging",
		"rc":               "cut a release candidate (vX.Y.Z-rc.N) instead of the final version",
		"no-hooks":         "skip the before/after hooks in .release.yaml for this run",
		"edit":             "open the generated release notes in your editor before writing",
	}
	for name, want := range wantFlagUsage {
		f := root.Flags().Lookup(name)
		if f == nil {
			f = root.PersistentFlags().Lookup(name)
		}
		if f == nil {
			t.Errorf("flag %q not found", name)
			continue
		}
		if f.Usage != want {
			t.Errorf("--%s usage = %q, want %q", name, f.Usage, want)
		}
	}

	help := helpReference()
	for _, want := range []string{
		"COMMANDS",
		"RELEASE FLAGS",
		"MENU",
		"Create a release: version + changelog + tag from commits since the last tag",
		"Conventional Commits drive the version: fix→patch, feat→minor, feat!/BREAKING→major.",
	} {
		if !strings.Contains(help, want) {
			t.Errorf("helpReference() missing %q", want)
		}
	}

	if got := i18n.T(i18n.VersionUpdateAvailable, "1.2.3"); got != "▲ v1.2.3 available — brew upgrade relio" {
		t.Errorf("i18n.T(VersionUpdateAvailable, ...) = %q", got)
	}
	if got := newVersionCmd().Short; got != "Print the Relio version" {
		t.Errorf("newVersionCmd().Short = %q", got)
	}
}
