// Package ui holds the shared lipgloss palette and the non-interactive views the
// CLI prints (banner, plan preview, success summary).
package ui

import (
	"fmt"
	"math"
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

	irisDim  = lipgloss.Color("130") // darker orange for the iris rim / roundness
	irisMid  = lipgloss.Color("166") // mid orange, iris body between rim and highlight
	glintCol = lipgloss.Color("223") // pale catchlight

	orangeMark = lipgloss.NewStyle().Bold(true).Foreground(Orange)
	whiteMark  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("231"))
	tagMark    = lipgloss.NewStyle().Foreground(dimCol).Italic(true) // banner tagline, muted grey
	ruleDim    = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
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

// "RELI" of the wordmark, ANSI Shadow block style, rendered in white. The "O" is
// the orange reptile eye built separately (see eyeLines).
var wordReli = []string{
	`██████╗ ███████╗██╗     ██╗`,
	`██╔══██╗██╔════╝██║     ██║`,
	`██████╔╝█████╗  ██║     ██║`,
	`██╔══██╗██╔══╝  ██║     ██║`,
	`██║  ██║███████╗███████╗██║`,
	`╚═╝  ╚═╝╚══════╝╚══════╝╚═╝`,
}

// The "O" is a reptile eye. It is drawn as a small pixel grid and rendered with
// half-block characters (▀ ▄ █), so every text row carries two pixel rows —
// twice the vertical detail of a plain block glyph in the same height. The grid
// is generated from a few ellipses: a shaded orange iris (dim rim, mid body,
// bright core), a hairline lens-shaped vertical slit, and a catchlight.
const (
	eyeW = 15 // odd, so the slit falls on a single centre column
	eyeH = 12 // two pixel rows per text line -> 6 lines, matching wordReli
)

// eyePixels builds the eyeH×eyeW grid. Cell values: 0 background, 1 dim rim,
// 2 mid iris, 3 bright core, 4 catchlight.
func eyePixels() [][]byte {
	g := make([][]byte, eyeH)
	cx, cy := float64(eyeW-1)/2, float64(eyeH-1)/2
	rx, ry := float64(eyeW)*0.46, float64(eyeH)*0.5

	for y := 0; y < eyeH; y++ {
		g[y] = make([]byte, eyeW)
		for x := 0; x < eyeW; x++ {
			dx := (float64(x) - cx) / rx
			dy := (float64(y) - cy) / ry
			d := math.Sqrt(dx*dx + dy*dy)
			switch {
			case d > 1.0:
				g[y][x] = 0
			case d > 0.82:
				g[y][x] = 1
			case d > 0.45:
				g[y][x] = 2
			default:
				g[y][x] = 3
			}
		}
	}

	// Hairline vertical slit: one column wide, stopping short of the iris edge
	// so it reads as a pointed lens rather than a full bar.
	for y := 0; y < eyeH; y++ {
		t := (float64(y) - cy) / (ry * 0.82)
		if t*t >= 1 {
			continue
		}
		half := 0.7 * (1 - t*t)
		for x := 0; x < eyeW; x++ {
			if math.Abs(float64(x)-cx) <= half && g[y][x] != 0 {
				g[y][x] = 0
			}
		}
	}

	// Catchlight: a small cluster on the iris, upper-left of the slit.
	for _, p := range [][2]int{
		{int(cx) - 2, int(cy) - 3},
		{int(cx) - 1, int(cy) - 3},
		{int(cx) - 2, int(cy) - 2},
	} {
		if x, y := p[0], p[1]; x >= 0 && x < eyeW && y >= 0 && y < eyeH && g[y][x] != 0 {
			g[y][x] = 4
		}
	}
	return g
}

func eyeColor(v byte) lipgloss.Color {
	switch v {
	case 1:
		return irisDim
	case 2:
		return irisMid
	case 4:
		return glintCol
	default:
		return Orange
	}
}

// eyeLines renders the pixel grid to eyeH/2 half-block strings. A cell packs the
// pixel above and below it: ▀ for a lit top, ▄ for a lit bottom, █ when both
// match, and ▀ with a background colour when the two halves differ.
func eyeLines() []string {
	g := eyePixels()
	lines := make([]string, 0, eyeH/2)
	for y := 0; y < eyeH; y += 2 {
		top, bot := g[y], g[y+1]
		var b strings.Builder
		for x := 0; x < eyeW; x++ {
			t, d := top[x], bot[x]
			switch {
			case t == 0 && d == 0:
				b.WriteByte(' ')
			case d == 0:
				b.WriteString(lipgloss.NewStyle().Foreground(eyeColor(t)).Render("▀"))
			case t == 0:
				b.WriteString(lipgloss.NewStyle().Foreground(eyeColor(d)).Render("▄"))
			case t == d:
				b.WriteString(lipgloss.NewStyle().Foreground(eyeColor(t)).Render("█"))
			default:
				b.WriteString(lipgloss.NewStyle().Foreground(eyeColor(t)).Background(eyeColor(d)).Render("▀"))
			}
		}
		lines = append(lines, b.String())
	}
	return lines
}

func padRight(s string, w int) string {
	if n := w - utf8.RuneCountInString(s); n > 0 {
		return s + strings.Repeat(" ", n)
	}
	return s
}

var bannerCache = map[string]string{}

// BigBanner is the entry banner: the "RELIO" block wordmark (white "RELI",
// orange "O" drawn as a snake eye) stands on its own as the title; the tagline,
// a rule, the current version — with the newer version beside it when available
// (a bare "1.2.3") — and the author credit all sit flush left below it.
func BigBanner(version, available string) string {
	key := version + "\x00" + available
	if s, ok := bannerCache[key]; ok {
		return s
	}

	const (
		indent = "  "
		gap    = 0 // the eye grid already carries a blank edge column; no extra space
	)
	lw := 0
	for _, l := range wordReli {
		lw = max(lw, utf8.RuneCountInString(l))
	}
	eye := eyeLines()
	total := lw + gap + eyeW

	var b strings.Builder
	b.WriteString("\n")

	for i := range wordReli {
		row := indent + whiteMark.Render(padRight(wordReli[i], lw)) + strings.Repeat(" ", gap)
		if i < len(eye) {
			row += eye[i]
		}
		b.WriteString(row + "\n")
	}

	// Lower block: a muted grey tagline, a two-tone accent rule (a short orange
	// lead fading into a thin grey line), then one meta line with the version
	// and author. The update notice, when present, gets its own orange line.
	const accent = 6
	rule := orangeMark.Render(strings.Repeat("━", accent)) +
		ruleDim.Render(strings.Repeat("─", max(0, total-accent)))

	b.WriteString("\n")
	b.WriteString(indent + tagMark.Render(Tagline) + "\n")
	b.WriteString(indent + rule + "\n")
	b.WriteString(indent + Key.Render(versionLabel(version)) +
		Dim.Render("   ·   created by ") + author.Render(Author) + "\n")
	if available != "" {
		b.WriteString(indent + Key.Render("▲ v"+strings.TrimPrefix(available, "v")+" available") + "\n")
	}

	out := b.String()
	bannerCache[key] = out
	return out
}

// versionLabel formats the build version for display: "v1.2.3", or "dev build"
// for an unstamped local build.
func versionLabel(version string) string {
	v := strings.TrimSpace(version)
	if v == "" || v == "dev" {
		return "dev build"
	}
	return "v" + strings.TrimPrefix(v, "v")
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
	if vf := versionFiles(p); vf != "" {
		parts = append(parts, vf, "")
	}
	parts = append(parts, Dim.Render(fmt.Sprintf("%d commits since %s", len(p.Commits), commitBase(p))))
	return strings.Join(parts, "\n")
}

// versionFiles renders the compact "Version files" section for the preview:
// one "  <rel>   <old> → <new>" line per pending edit.
func versionFiles(p release.Plan) string {
	if len(p.VersionChanges) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(group.Render("Version files") + "\n")
	for _, c := range p.VersionChanges {
		b.WriteString("  " + c.Rel + "   " + Dim.Render(c.Old+" → "+c.New) + "\n")
	}
	return strings.TrimRight(b.String(), "\n")
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
