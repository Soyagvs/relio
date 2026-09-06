// Package ui holds the shared lipgloss palette and the non-interactive views the
// CLI prints (banner, plan preview, success summary).
package ui

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"

	"github.com/soyagvs/go-release/internal/changelog"
	"github.com/soyagvs/go-release/internal/release"
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
	purpleMark = lipgloss.NewStyle().Bold(true).Foreground(Purple)
	author     = lipgloss.NewStyle().Bold(true).Foreground(Orange)
	group      = lipgloss.NewStyle().Bold(true).Foreground(Orange)
	rule       = lipgloss.NewStyle().Foreground(Purple)
	hash       = lipgloss.NewStyle().Foreground(Purple)

	box = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(Purple).
		Padding(0, 2)
)

// AppName and Author are shown on the big entry banner.
const (
	AppName = "Go Release"
	Author  = "SOYAGVS"
)

// wordmarkGo / wordmarkRelease are the two halves of the ANSI Shadow title,
// coloured separately (orange "Go", purple "Release"). Each slice is one row;
// rows are padded to equal width at render time.
var (
	wordmarkGo = []string{
		` ██████╗  ██████╗ `,
		`██╔════╝ ██╔═══██╗`,
		`██║  ███╗██║   ██║`,
		`██║   ██║██║   ██║`,
		`╚██████╔╝╚██████╔╝`,
		` ╚═════╝  ╚═════╝ `,
	}
	wordmarkRelease = []string{
		`██████╗ ███████╗██╗     ███████╗ █████╗ ███████╗███████╗`,
		`██╔══██╗██╔════╝██║     ██╔════╝██╔══██╗██╔════╝██╔════╝`,
		`██████╔╝█████╗  ██║     █████╗  ███████║███████╗█████╗  `,
		`██╔══██╗██╔══╝  ██║     ██╔══╝  ██╔══██║╚════██║██╔══╝  `,
		`██║  ██║███████╗███████╗███████╗██║  ██║███████║███████╗`,
		`╚═╝  ╚═╝╚══════╝╚══════╝╚══════╝╚═╝  ╚═╝╚══════╝╚══════╝`,
	}
)

func padRight(s string, w int) string {
	if n := w - utf8.RuneCountInString(s); n > 0 {
		return s + strings.Repeat(" ", n)
	}
	return s
}

var bannerCache = map[string]string{}

// BigBanner is the entry banner: the two-tone wordmark, a rule, and the author
// credit centred beneath the name. Printed once when the menu opens.
func BigBanner(version string) string {
	if s, ok := bannerCache[version]; ok {
		return s
	}

	const indent = "  "
	gw, rw := 0, 0
	for _, l := range wordmarkGo {
		gw = max(gw, utf8.RuneCountInString(l))
	}
	for _, l := range wordmarkRelease {
		rw = max(rw, utf8.RuneCountInString(l))
	}
	total := gw + 1 + rw

	var b strings.Builder
	b.WriteString("\n")
	for i := range wordmarkGo {
		b.WriteString(indent +
			orangeMark.Render(padRight(wordmarkGo[i], gw)) + " " +
			purpleMark.Render(padRight(wordmarkRelease[i], rw)) + "\n")
	}
	b.WriteString(indent +
		orangeMark.Render(strings.Repeat("━", gw+1)) +
		rule.Render(strings.Repeat("━", rw)) + "\n")

	credit := "created by " + Author
	lead := max(0, (total-utf8.RuneCountInString(credit))/2)
	b.WriteString(indent + strings.Repeat(" ", lead) + Dim.Render("created by ") + author.Render(Author))
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

// Notes renders the grouped release notes as indented bullet lists.
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
			marker := Dim.Render("      •")
			if it.Hash != "" {
				marker = hash.Render(fmt.Sprintf("%7s", it.Hash))
			}
			b.WriteString("  " + marker + "  " + it.Text + "\n")
		}
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

// ReleaseHeader is the "<project> -- release / <version> / <meta>" block.
func ReleaseHeader(project, version, meta string) string {
	return Key.Render(project+" -- release") + "\n" +
		Ok.Render(version) + "\n" +
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
