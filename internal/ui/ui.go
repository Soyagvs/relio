// Package ui holds the shared lipgloss palette and the non-interactive views the
// CLI prints (banner, plan preview, success summary).
package ui

import (
	"fmt"
	"io"
	"math"
	"os"
	"runtime"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"

	"github.com/soyagvs/relio/internal/changelog"
	"github.com/soyagvs/relio/internal/i18n"
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

// AppName and Author are shown on the big entry banner. RepoURL is the project's
// GitHub home ("Soyagvs" is the account name — capital S is intentional).
const (
	AppName = "Relio"
	Author  = "SOYAGVS"
	RepoURL = "github.com/Soyagvs/relio"
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

// restGlint is the eye's resting catchlight position — upper-left-ish, roughly
// where the static catchlight sat before the one-shot sweep was added.
const restGlint = 0.30

// eyePixels builds the eyeH×eyeW grid. Cell values: 0 background, 1 dim rim,
// 2 mid iris, 3 bright core, 4 catchlight. glint is a normalized sweep position:
// the catchlight rides left→right across the upper iris as it goes 0→1, and is
// off-frame at the extremes.
func eyePixels(glint float64) [][]byte {
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

	// Catchlight: a bright spot that rides left→right across the upper iris as
	// glint sweeps 0→1. Off-frame at the extremes; a brief bloom near mid-sweep,
	// a tighter dot near the ends. Painted only where the cell is already on the
	// iris, so the light never spills past the rim.
	if glint > 0.02 && glint < 0.98 {
		gx := 1.5 + glint*(float64(eyeW)-3.0)
		gy := cy - ry*0.5
		radius := 1.4
		switch {
		case glint >= 0.35 && glint <= 0.65:
			radius = 1.8
		case glint < 0.15 || glint > 0.85:
			radius = 1.0
		}
		for y := 0; y < eyeH; y++ {
			for x := 0; x < eyeW; x++ {
				if g[y][x] == 0 {
					continue
				}
				gdx, gdy := float64(x)-gx, float64(y)-gy
				if math.Sqrt(gdx*gdx+gdy*gdy) <= radius {
					g[y][x] = 4
				}
			}
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
func eyeLines(glint float64) []string {
	g := eyePixels(glint)
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

const (
	bannerIndent = "  "
	bannerGap    = 0 // the eye grid already carries a blank edge column; no extra space
	bannerAccent = 6 // orange lead length of the two-tone rule
)

// wordmarkWidth is the rune width of the widest wordReli row (all six are equal
// today), computed once.
var wordmarkWidth = func() int {
	w := 0
	for _, l := range wordReli {
		w = max(w, utf8.RuneCountInString(l))
	}
	return w
}()

// bannerWordmark builds the leading newline plus the six wordmark rows: the
// white "RELI" blocks with the orange eye — its catchlight at sweep position
// glint — beside them.
func bannerWordmark(glint float64) string {
	eye := eyeLines(glint)
	var b strings.Builder
	b.WriteString("\n")
	for i := range wordReli {
		row := bannerIndent + whiteMark.Render(padRight(wordReli[i], wordmarkWidth)) + strings.Repeat(" ", bannerGap)
		if i < len(eye) {
			row += eye[i]
		}
		b.WriteString(row + "\n")
	}
	return b.String()
}

// bannerLower builds the block under the wordmark: a blank line, a muted grey
// tagline, a two-tone accent rule (a short orange lead fading into a thin grey
// line), one meta line with the version and author, the repo URL, and — when
// available is set — an orange update notice on its own line.
func bannerLower(version, available string) string {
	total := wordmarkWidth + bannerGap + eyeW
	rule := orangeMark.Render(strings.Repeat("━", bannerAccent)) +
		ruleDim.Render(strings.Repeat("─", max(0, total-bannerAccent)))

	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(bannerIndent + tagMark.Render(Tagline) + "\n")
	b.WriteString(bannerIndent + rule + "\n")
	b.WriteString(bannerIndent + Key.Render(versionLabel(version)) +
		Dim.Render("   ·   created by ") + author.Render(Author) + "\n")
	b.WriteString(bannerIndent + Dim.Render(RepoURL) + "\n")
	if available != "" {
		b.WriteString(bannerIndent + Key.Render("▲ v"+strings.TrimPrefix(available, "v")+" available") + "\n")
	}
	return b.String()
}

// BigBanner is the entry banner: the "RELIO" block wordmark (white "RELI",
// orange "O" drawn as a snake eye) stands on its own as the title; the tagline,
// a rule, the current version — with the newer version beside it when available
// (a bare "1.2.3") — the repo URL, and the author credit all sit flush left
// below it.
func BigBanner(version, available string) string {
	key := version + "\x00" + available
	if s, ok := bannerCache[key]; ok {
		return s
	}
	out := BannerFrame(version, available, restGlint)
	bannerCache[key] = out
	return out
}

// introFrameDelay is the pause between sweep frames. A package var so tests can
// zero it.
var introFrameDelay = 250 * time.Millisecond

// BannerIntroFrameDelay is the pause between animated banner frames.
func BannerIntroFrameDelay() time.Duration { return introFrameDelay }

// BannerAnimationAllowed reports whether the environment allows banner animation.
func BannerAnimationAllowed(animate bool) bool {
	return animate && runtime.GOOS != "windows" &&
		os.Getenv("NO_COLOR") == "" && os.Getenv("RELIO_NO_ANIM") == ""
}

// BannerIntroFrames returns the glint positions for the non-blocking intro.
func BannerIntroFrames() []float64 {
	const sweepSteps = 9
	schedule := make([]float64, 0, sweepSteps+3)
	schedule = append(schedule, 0)
	for i := 0; i < sweepSteps; i++ {
		t := float64(i) / float64(sweepSteps-1)
		schedule = append(schedule, 0.06+t*(1.02-0.06))
	}
	return append(schedule, 0.55, restGlint)
}

// BannerFrame renders the full banner with the eye catchlight at glint.
func BannerFrame(version, available string, glint float64) string {
	return bannerWordmark(glint) + bannerLower(version, available)
}

// BannerIntro prints the entry banner. When animate is true and the environment
// allows it, the eye plays a one-shot light sweep before the rest of the banner
// prints; otherwise it is identical to BigBanner. The sweep is skipped on
// Windows and when NO_COLOR or RELIO_NO_ANIM is set.
func BannerIntro(w io.Writer, version, available string, animate bool) {
	if !BannerAnimationAllowed(animate) {
		fmt.Fprint(w, BigBanner(version, available))
		return
	}

	// The eye with no catchlight; the cursor ends on the line below row 6.
	frames := BannerIntroFrames()
	fmt.Fprint(w, bannerWordmark(frames[0]))

	for _, g := range frames[1:] {
		fmt.Fprint(w, "\x1b[6A") // back up over the six wordmark rows
		// bannerWordmark starts with "\n", so field 0 is empty; rows 1..6 are the
		// wordmark rows.
		rows := strings.Split(bannerWordmark(g), "\n")
		for _, row := range rows[1:7] {
			fmt.Fprint(w, "\r\x1b[2K"+row+"\n")
		}
		time.Sleep(introFrameDelay)
	}

	// The eye is now at restGlint and the cursor is below the six rows.
	fmt.Fprint(w, bannerLower(version, available))
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

// PlanBox renders the version summary card shown before confirmation. This is
// interactive chrome, not artifact vocabulary: it routes through i18n.T and
// IS localized — see internal/ui/artifact_invariance_test.go.
func PlanBox(p release.Plan) string {
	row := func(k, v string) string {
		return Dim.Render(fmt.Sprintf("%-16s", k)) + v
	}
	bumpLabel := p.Bump.String()
	if p.BumpForced {
		bumpLabel += Dim.Render(i18n.T(i18n.PlanForced))
	}
	nextLine := Ok.Render(p.Next.String())
	if p.Prerelease {
		nextLine += Dim.Render(i18n.T(i18n.PlanPrerelease))
	}
	if p.Finalizing {
		nextLine += Dim.Render(i18n.T(i18n.PlanFinalize))
	}
	lines := []string{
		row(i18n.T(i18n.PlanCurrentVersion), p.Current.String()),
		row(i18n.T(i18n.PlanDetectedChange), bumpLabel),
		row(i18n.T(i18n.PlanNextVersion), nextLine),
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
		b.WriteString(group.Render(g.Label()) + "\n")
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
	if h := hooks(p); h != "" {
		parts = append(parts, h, "")
	}
	parts = append(parts, Dim.Render(i18n.T(i18n.PlanCommitsSince, len(p.Commits), commitBase(p))))
	return strings.Join(parts, "\n")
}

// hooks renders a terse summary of the configured before/after release hooks:
// the command itself when there is one, "N commands" when there are several.
func hooks(p release.Plan) string {
	before, after := p.Config.Release.Hooks.Before, p.Config.Release.Hooks.After
	if len(before) == 0 && len(after) == 0 {
		return ""
	}
	summary := func(cmds []string) string {
		if len(cmds) == 1 {
			return cmds[0]
		}
		return fmt.Sprintf("%d commands", len(cmds))
	}
	var b strings.Builder
	b.WriteString(group.Render("hooks") + "\n")
	if len(before) > 0 {
		b.WriteString("  " + Dim.Render("before: "+summary(before)) + "\n")
	}
	if len(after) > 0 {
		b.WriteString("  " + Dim.Render("after: "+summary(after)) + "\n")
	}
	return strings.TrimRight(b.String(), "\n")
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
