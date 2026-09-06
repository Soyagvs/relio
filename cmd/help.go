package cmd

import (
	"fmt"
	"strings"

	"github.com/soyagvs/go-release/internal/ui"
)

// reference is the full command + flag listing shown by the menu's Help entry.
type refRow struct{ name, desc string }

var (
	refCommands = []refRow{
		{"go-release", "Create a release: version + changelog + tag from commits since the last tag"},
		{"go-release init", "Create .release.yaml in the current repo (configuration only, never secrets)"},
		{"go-release post", "Print copy-paste release text for social posts (text on stdout only)"},
		{"go-release auth", "GitHub login — placeholder, lands in v0.2.0"},
		{"go-release version", "Print the Go Release version"},
	}
	refReleaseFlags = []refRow{
		{"--patch / --minor / --major", "Force the version bump instead of inferring it from the commits"},
		{"-y, --yes", "Skip the menu and the confirmation (required in CI or a non-interactive shell)"},
		{"--no-changelog", "Do not modify the changelog file"},
		{"--no-tag", "Do not create the git tag"},
		{"-C, --dir <path>", "Run as if Go Release was started in <path>"},
	}
	refPostFlags = []refRow{
		{"--format minimal", "Same output as the releases browser: project, version, date, commits, grouped notes (default)"},
		{"--format technical", "Terse bullet list, for a changelog or a dev channel"},
		{"--format casual", "Loose tone: \"proj v1.4.0 is out. → …\""},
		{"--format changelog", "The exact section that goes into CHANGELOG.md"},
	}
	refMenu = []refRow{
		{"Create a release", "Same as running `go-release` with no arguments"},
		{"Releases", "List versions, read a version's notes, or delete one (git tag + changelog section)"},
		{"Release text", "Pick a post format and print copy-paste text (same as `go-release post`)"},
		{"GitHub auth", "Preview of the v0.2.0 GitHub integration"},
		{"Help", "This screen"},
		{"Exit", "Leave Go Release"},
	}
)

func helpReference() string {
	var b strings.Builder
	b.WriteString(ui.Banner("", version) + "\n\n")

	section := func(title string, rows []refRow) {
		b.WriteString(ui.Key.Render(strings.ToUpper(title)) + "\n")
		width := 0
		for _, r := range rows {
			if len(r.name) > width {
				width = len(r.name)
			}
		}
		for _, r := range rows {
			b.WriteString("  " + fmt.Sprintf("%-*s", width, r.name) + "  " + ui.Dim.Render(r.desc) + "\n")
		}
		b.WriteString("\n")
	}

	section("Commands", refCommands)
	section("Release flags", refReleaseFlags)
	section("post flags", refPostFlags)
	section("Menu", refMenu)

	b.WriteString(ui.Dim.Render("Conventional Commits drive the version: fix→patch, feat→minor, feat!/BREAKING→major."))
	return b.String()
}
