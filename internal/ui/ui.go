// Package ui holds the shared lipgloss palette and the non-interactive views the
// CLI prints (banner, plan preview, success summary).
package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/soyagvs/go-release/internal/changelog"
	"github.com/soyagvs/go-release/internal/release"
)

// Palette. Colors are ANSI-256 so they degrade gracefully on limited terminals.
var (
	accent  = lipgloss.Color("42")  // green
	accent2 = lipgloss.Color("39")  // cyan
	warnCol = lipgloss.Color("214") // amber
	dimCol  = lipgloss.Color("245")

	Title = lipgloss.NewStyle().Bold(true).Foreground(accent)
	Dim   = lipgloss.NewStyle().Foreground(dimCol)
	Key   = lipgloss.NewStyle().Foreground(accent2)
	Warn  = lipgloss.NewStyle().Foreground(warnCol)
	Ok    = lipgloss.NewStyle().Foreground(accent).Bold(true)

	group = lipgloss.NewStyle().Bold(true).Foreground(accent2)

	box = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(dimCol).
		Padding(0, 2)

	wordmark = lipgloss.NewStyle().Bold(true).Foreground(accent)

	bannerBox = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(accent).
			Padding(1, 4).
			Align(lipgloss.Center)
)

// AppName and Author are shown on the big entry banner.
const (
	AppName = "Go Release"
	Author  = "SOYAGVS"
)

// asciiWordmark is the figlet-style "GO RELEASE" title.
const asciiWordmark = ` ██████╗  ██████╗    ██████╗ ███████╗██╗     ███████╗ █████╗ ███████╗███████╗
██╔════╝ ██╔═══██╗   ██╔══██╗██╔════╝██║     ██╔════╝██╔══██╗██╔════╝██╔════╝
██║  ███╗██║   ██║   ██████╔╝█████╗  ██║     █████╗  ███████║███████╗█████╗
██║   ██║██║   ██║   ██╔══██╗██╔══╝  ██║     ██╔══╝  ██╔══██║╚════██║██╔══╝
╚██████╔╝╚██████╔╝   ██║  ██║███████╗███████╗███████╗██║  ██║███████║███████╗
 ╚═════╝  ╚═════╝    ╚═╝  ╚═╝╚══════╝╚══════╝╚══════╝╚═╝  ╚═╝╚══════╝╚══════╝`

// BigBanner is the full entry banner: the wordmark, tagline, and author credit.
// It is printed once when the interactive menu opens.
func BigBanner(version string) string {
	inner := wordmark.Render(asciiWordmark) + "\n\n" +
		Dim.Render("from finished code to a published release")
	framed := bannerBox.Render(inner)

	credit := Dim.Render("created by ") + Key.Render(Author)
	if version != "" {
		credit += Dim.Render("  ·  " + version)
	}
	return framed + "\n" + lipgloss.NewStyle().PaddingLeft(2).Render(credit)
}

// Banner is the small header printed at the top of a command run.
func Banner(project, version string) string {
	mark := Title.Render("⬢ " + AppName)
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
			b.WriteString("  " + Dim.Render("•") + " " + it + "\n")
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
