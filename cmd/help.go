package cmd

import (
	"fmt"
	"strings"

	"github.com/soyagvs/relio/internal/ui"
)

// reference is the full command + flag listing shown by the menu's Help entry.
type refRow struct{ name, desc string }

var (
	refCommands = []refRow{
		{"relio", "Create a release: version + changelog + tag from commits since the last tag"},
		{"relio status", "Show what's unreleased since the last tag and the version it suggests"},
		{"relio stats", "Relio's public GitHub download stats (read-only, no telemetry)"},
		{"relio init", "Create .release.yaml in the current repo (configuration only, never secrets)"},
		{"relio post", "Print copy-paste release text for social posts (text on stdout only)"},
		{"relio image", "Make a release card image — save it, upload it for a link, or both (--shape, --theme, --hash, --upload, --link-only)"},
		{"relio auth", "GitHub login — placeholder, lands in v0.2.0"},
		{"relio version", "Print the Relio version"},
	}
	refReleaseFlags = []refRow{
		{"--patch / --minor / --major", "Force the version bump instead of inferring it from the commits"},
		{"-y, --yes", "Skip the menu and the confirmation (required in CI or a non-interactive shell)"},
		{"--no-changelog", "Do not modify the changelog file"},
		{"--no-tag", "Do not create the git tag"},
		{"--no-hash", "Hide the commit hash on each release-note line"},
		{"-C, --dir <path>", "Run as if Relio was started in <path>"},
	}
	refPostFlags = []refRow{
		{"--format minimal", "Same output as the releases browser: project, version, date, commits, grouped notes (default)"},
		{"--format social", "Shortest: \"Project -- Release\", version · date · time, then \"type  description\" lines"},
		{"--format technical", "Terse bullet list, for a changelog or a dev channel"},
		{"--format casual", "Loose tone: \"proj v1.4.0 is out. → …\""},
		{"--format changelog", "The exact section that goes into CHANGELOG.md"},
	}
	refImageFlags = []refRow{
		{"--shape", "horizontal (1200×630) | vertical (1080×1920) | square (1080×1080)"},
		{"--theme", "orange (default) | green | purple accent"},
		{"--hash", "show the commit hash on each line"},
		{"--upload", "also upload to a temp host (litterbox 72h) and print a link + QR"},
		{"--link-only", "upload for a link + QR without writing a local file"},
	}
	refMenu = []refRow{
		{"Status", "What's unreleased and the suggested version (same as `relio status`)"},
		{"Create a release", "Same as running `relio` with no arguments"},
		{"Releases", "List versions, read a version's notes, or delete one (git tag + changelog section)"},
		{"Release text", "Pick a post format and print copy-paste text (same as `relio post`)"},
		{"Release image", "Pick a release + shape, then save the card, upload it for a link, or both (same as `relio image`)"},
		{"GitHub auth", "Preview of the v0.2.0 GitHub integration"},
		{"Help", "This screen"},
		{"Exit", "Leave Relio"},
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
	section("image flags", refImageFlags)
	section("Menu", refMenu)

	b.WriteString(ui.Dim.Render("Conventional Commits drive the version: fix→patch, feat→minor, feat!/BREAKING→major."))
	return b.String()
}
