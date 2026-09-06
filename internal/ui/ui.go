// Package ui holds the shared lipgloss palette and the non-interactive views the
// CLI prints (banner, plan preview, success summary).
package ui

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"

	"github.com/soyagvs/relio/internal/changelog"
	"github.com/soyagvs/relio/internal/release"
)

// Brand palette: orange + purple, red reserved for failures. 256-colour indices
// so the look holds up on terminals without truecolor.
var (
	Orange = lipgloss.Color("208")
	Purple = lipgloss.Color("135")
	redCol = lipgloss.Color("203")
	dimCol = lipgloss.Color("245")

	Title = lipgloss.NewStyle().Bold(true).Foreground(Purple)
	Dim   = lipgloss.NewStyle().Foreground(dimCol)
	Key   = lipgloss.NewStyle().Foreground(Orange)
	Warn  = lipgloss.NewStyle().Foreground(redCol)
	Ok    = lipgloss.NewStyle().Foreground(Purple).Bold(true)

	orangeMark = lipgloss.NewStyle().Bold(true).Foreground(Orange)
	whiteMark  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("231"))
	glintMark  = lipgloss.NewStyle().Foreground(lipgloss.Color("223")) // snake-eye catchlight
	author     = lipgloss.NewStyle().Bold(true).Foreground(Orange)
	group      = lipgloss.NewStyle().Bold(true).Foreground(Orange)
	hash       = lipgloss.NewStyle().Foreground(Purple)

	box = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(Purple).
		Padding(0, 2)
)

// AppName and Author are shown on the big entry banner.
const (
	AppName = "Relio"
	Author  = "SOYAGVS"
)

// Tagline sits under the wordmark, in purple.
const Tagline = "turn commits into releases"

// The wordmark is "RELIO" in the ANSI Shadow block style, split so "RELI"
// renders in white and "O" in orange. The "O" carries a vertical slit so it
// reads as a snake eye. Rows are padded to equal width at render time.
var (
	wordReli = []string{
		`██████╗ ███████╗██╗     ██╗`,
		`██╔══██╗██╔════╝██║     ██║`,
		`██████╔╝█████╗  ██║     ██║`,
		`██╔══██╗██╔══╝  ██║     ██║`,
		`██║  ██║███████╗███████╗██║`,
		`╚═╝  ╚═╝╚══════╝╚══════╝╚═╝`,
	}
	// A solid orange eyeball with a lens-shaped vertical slit carved out (the
	// dark terminal background) and a small catchlight — a snake eye.
	wordO = []string{
		` ███████ `,
		`████ ████`,
		`███ ▪ ███`,
		`███   ███`,
		`████ ████`,
		` ███████ `,
	}
)

// renderO colours the eyeball orange and the catchlight pale.
func renderO(row string) string {
	var b strings.Builder
	for _, r := range row {
		switch r {
		case ' ':
			b.WriteRune(' ')
		case '▪':
			b.WriteString(glintMark.Render("▪"))
		default:
			b.WriteString(orangeMark.Render(string(r)))
		}
	}
	return b.String()
}

func padRight(s string, w int) string {
	if n := w - utf8.RuneCountInString(s); n > 0 {
		return s + strings.Repeat(" ", n)
	}
	return s
}

var bannerCache = map[string]string{}

// BigBanner is the entry banner: the "RELIO" block wordmark (white "RELI",
// orange "O" drawn as a snake eye), the tagline, a rule, and the author credit.
func BigBanner(version string) string {
	if s, ok := bannerCache[version]; ok {
		return s
	}

	const indent = "  "
	lw, ow := 0, 0
	for _, l := range wordReli {
		lw = max(lw, utf8.RuneCountInString(l))
	}
	for _, l := range wordO {
		ow = max(ow, utf8.RuneCountInString(l))
	}
	total := lw + ow

	// lead centres a line of the given visible width within the wordmark.
	lead := func(visibleWidth int) string {
		return strings.Repeat(" ", max(0, (total-visibleWidth)/2))
	}

	var b strings.Builder
	b.WriteString("\n")

	for i := range wordReli {
		b.WriteString(indent +
			whiteMark.Render(padRight(wordReli[i], lw)) +
			renderO(wordO[i]) + "\n")
	}
	b.WriteString(indent + lead(utf8.RuneCountInString(Tagline)) + Title.Render(Tagline) + "\n")
	b.WriteString(indent + orangeMark.Render(strings.Repeat("━", total)) + "\n")

	creditText := "created by " + Author
	if version != "" {
		creditText += "   " + version
	}
	b.WriteString(indent + lead(utf8.RuneCountInString(creditText)) +
		Dim.Render("created by ") + author.Render(Author))
	if version != "" {
		b.WriteString(Dim.Render("   " + version))
	}
	b.WriteString("\n")

	out := b.String()
	bannerCache[version] = out
	return out
}

// Banner is the small header printed at the top of a command run.
func Banner(project, version string) string {
	mark := Key.Render("⬢ ") + Title.Render(AppName)
	meta := Dim.Render(version)
	if project != "" {
		meta = Dim.Render(version + "  ·  " + project)
	}
	return mark + "  " + meta
}

// PlanBox renders the version summary card shown before confirmation.
func PlanBox(p release.Plan) string {
	row := func(k, v string) string {
		return Dim.Render(fmt.Sprintf("%-16s", k)) + v
	}
	bumpLabel := p.Bump.String()
	if p.BumpForced {
		bumpLabel += Dim.Render(" (forced)")
	}
	lines := []string{
		row("Current version", p.Current.String()),
		row("Detected change", bumpLabel),
		row("Next version", Ok.Render(p.Next.String())),
	}
	return box.Render(strings.Join(lines, "\n"))
}

// HideHashes suppresses the per-line commit hash in Notes (set by --no-hash).
var HideHashes bool

// Notes renders the grouped release notes as indented lists, "hash  text" per
// line unless HideHashes is set.
func Notes(n changelog.Notes) string {
	order := []changelog.Group{
		changelog.Added, changelog.Changed, changelog.Deprecated,
		changelog.Removed, changelog.Fixed, changelog.Security,
	}
	var b strings.Builder
	for _, g := range order {
		items := n.Groups[g]
		if len(items) == 0 {
			continue
		}
		b.WriteString(group.Render(string(g)) + "\n")
		for _, it := range items {
			if HideHashes || it.Hash == "" {
				b.WriteString("  " + Dim.Render("•") + " " + it.Text + "\n")
				continue
			}
			b.WriteString("  " + hash.Render(fmt.Sprintf("%7s", it.Hash)) + "  " + it.Text + "\n")
		}
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

// ReleaseMeta is the "<date> · <time> · N commits" line. datetime is
// "2006-01-02 15:04".
func ReleaseMeta(datetime string, commits int) string {
	return fmt.Sprintf("%s · %d commits", strings.Replace(datetime, " ", " · ", 1), commits)
}

// ReleaseHeader is the block above a version's notes:
//
//	relio -- release
//
//	<project> · <version>
//	<meta>
func ReleaseHeader(project, version, meta string) string {
	return Key.Render(strings.ToLower(AppName)+" -- release") + "\n\n" +
		Key.Render(project) + Dim.Render(" · ") + Ok.Render(version) + "\n" +
		Dim.Render(meta)
}

// ReleaseText renders the canonical release summary shared by the releases
// browser and `post --format minimal`: the header plus the grouped notes with
// commit hashes. Colours are stripped automatically when stdout is not a TTY.
func ReleaseText(project, version, meta string, notes changelog.Notes) string {
	body := Notes(notes)
	if strings.TrimSpace(body) == "" {
		body = Dim.Render("(no user-facing changes)")
	}
	return ReleaseHeader(project, version, meta) + "\n\n" + body
}

// Markdownish adds colour to a plain Keep a Changelog block for terminal display
// without changing the text (it stays a valid changelog section).
func Markdownish(s string) string {
	var b strings.Builder
	for _, ln := range strings.Split(s, "\n") {
		switch {
		case strings.HasPrefix(ln, "## "):
			b.WriteString(Title.Render(ln))
		case strings.HasPrefix(ln, "### "):
			b.WriteString(group.Render(ln))
		case strings.HasPrefix(ln, "- "):
			b.WriteString(Dim.Render("- ") + strings.TrimPrefix(ln, "- "))
		default:
			b.WriteString(ln)
		}
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

// PlanView is the full non-interactive preview: card + notes + commit count.
func PlanView(p release.Plan) string {
	parts := []string{PlanBox(p), ""}
	if notes := Notes(p.Notes); notes != "" {
		parts = append(parts, notes, "")
	}
	parts = append(parts, Dim.Render(fmt.Sprintf("%d commits since %s", len(p.Commits), commitBase(p))))
	return strings.Join(parts, "\n")
}

func commitBase(p release.Plan) string {
	if p.Current.String() == "v0.0.0" {
		return "the beginning"
	}
	return p.Current.String()
}

// Success renders the checklist printed after a release is applied.
func Success(lines []string) string {
	var b strings.Builder
	for _, ln := range lines {
		b.WriteString(Ok.Render("✓") + " " + ln + "\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

// Info prints a dimmed informational line with a leading marker.
func Info(s string) string { return Dim.Render("· " + s) }
