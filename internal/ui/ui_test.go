package ui

import (
	"bytes"
	"regexp"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/soyagvs/relio/internal/changelog"
	"github.com/soyagvs/relio/internal/config"
	"github.com/soyagvs/relio/internal/i18n"
	"github.com/soyagvs/relio/internal/release"
	"github.com/soyagvs/relio/internal/semver"
	"github.com/soyagvs/relio/internal/versionfile"
)

var ansiRE = regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]`)

func stripANSI(s string) string { return ansiRE.ReplaceAllString(s, "") }

// TestTruncate covers the shared implementation promoted from the menu
// package so both the menu and the Settings screen truncate long labels and
// descriptions identically.
func TestTruncate(t *testing.T) {
	tests := []struct {
		in       string
		maxRunes int
		want     string
	}{
		{"short", 10, "short"},
		{"exactly10!", 10, "exactly10!"},
		{"this is too long", 8, "this is…"},
		{"x", 1, "x"},
		{"toolong", 1, "…"},
		{"anything", 0, ""},
		{"anything", -1, ""},
		{"日本語のテキスト", 4, "日本語…"},
	}
	for _, tt := range tests {
		if got := Truncate(tt.in, tt.maxRunes); got != tt.want {
			t.Errorf("Truncate(%q, %d) = %q, want %q", tt.in, tt.maxRunes, got, tt.want)
		}
	}
}

func TestBigBannerHasRepoURL(t *testing.T) {
	out := BigBanner("v1.2.3", "")
	for _, want := range []string{"github.com/Soyagvs/relio", "created by", Tagline, "█"} {
		if !strings.Contains(out, want) {
			t.Errorf("BigBanner output missing %q:\n%s", want, out)
		}
	}
}

func TestBigBannerStartsWithNewline(t *testing.T) {
	out := BigBanner("v9.9.9", "")
	if out == "" || out[0] != '\n' {
		t.Fatalf("BigBanner must start with a newline")
	}
}

func TestEyeGlintMoves(t *testing.T) {
	count := func(g [][]byte, want byte) int {
		n := 0
		for _, row := range g {
			for _, v := range row {
				if v == want {
					n++
				}
			}
		}
		return n
	}
	if got := count(eyePixels(-1), 4); got != 0 {
		t.Errorf("eyePixels(-1) should paint no catchlight, got %d cells == 4", got)
	}
	rest := eyePixels(restGlint)
	if got := count(rest, 4); got < 1 {
		t.Errorf("eyePixels(restGlint) should paint a catchlight, got %d cells == 4", got)
	}
	if got := count(rest, 3); got < 1 {
		t.Errorf("eyePixels(restGlint) lost its bright core (no cell == 3)")
	}
	meanX := func(glint float64) float64 {
		g := eyePixels(glint)
		var sum, n float64
		for _, row := range g {
			for x, v := range row {
				if v == 4 {
					sum += float64(x)
					n++
				}
			}
		}
		if n == 0 {
			t.Fatalf("eyePixels(%v) has no value-4 cells to average", glint)
		}
		return sum / n
	}
	if hi, lo := meanX(0.8), meanX(0.15); hi <= lo {
		t.Errorf("catchlight mean x at glint 0.8 (%v) should exceed glint 0.15 (%v)", hi, lo)
	}
}

func TestEyeLinesStableSize(t *testing.T) {
	glints := []float64{-1, 0.3, 0.8}
	base := eyeLines(glints[0])
	for _, g := range glints[1:] {
		lines := eyeLines(g)
		if len(lines) != len(base) {
			t.Fatalf("eyeLines(%v): %d lines, eyeLines(%v): %d lines", g, len(lines), glints[0], len(base))
		}
		for i := range lines {
			if w, bw := utf8.RuneCountInString(stripANSI(lines[i])), utf8.RuneCountInString(stripANSI(base[i])); w != bw {
				t.Errorf("eyeLines(%v) line %d visible width %d != %d", g, i, w, bw)
			}
		}
	}
}

func TestBannerIntroStaticWhenNotAnimating(t *testing.T) {
	var b bytes.Buffer
	BannerIntro(&b, "v1.2.3", "", false)
	if got, want := b.String(), BigBanner("v1.2.3", ""); got != want {
		t.Errorf("BannerIntro(animate=false) must equal BigBanner\n got: %q\nwant: %q", got, want)
	}
}

func TestBannerIntroRespectsNoAnimEnv(t *testing.T) {
	t.Setenv("RELIO_NO_ANIM", "1")
	var b bytes.Buffer
	BannerIntro(&b, "v1.2.3", "", true)
	if got, want := b.String(), BigBanner("v1.2.3", ""); got != want {
		t.Errorf("BannerIntro must be static when RELIO_NO_ANIM is set")
	}
}

func TestBannerIntroAnimatedEndsAtStaticBanner(t *testing.T) {
	saved := introFrameDelay
	introFrameDelay = 0
	defer func() { introFrameDelay = saved }()
	t.Setenv("NO_COLOR", "")
	t.Setenv("RELIO_NO_ANIM", "")

	var b bytes.Buffer
	BannerIntro(&b, "v1.2.3", "v1.3.0", true)
	clean := stripANSI(b.String())
	for _, want := range []string{Tagline, "created by", "github.com/Soyagvs/relio", "▲ v1.3.0 available", "█"} {
		if !strings.Contains(clean, want) {
			t.Errorf("animated banner (cleaned) missing %q:\n%s", want, clean)
		}
	}
}

func TestPlanViewListsVersionFiles(t *testing.T) {
	v15 := semver.Version{Major: 1, Minor: 5, Prefix: "v"}
	v16 := semver.Version{Major: 1, Minor: 6, Prefix: "v"}
	p := release.Plan{
		Current: v15,
		Next:    v16,
		Bump:    semver.Minor,
		VersionChanges: []versionfile.Change{
			{Rel: "package.json", Old: "1.5.0", New: "1.6.0"},
			{Rel: "VERSION", Old: "1.5.0", New: "1.6.0"},
		},
	}
	out := PlanView(p)
	if !strings.Contains(out, "Version files") {
		t.Errorf("missing 'Version files' section:\n%s", out)
	}
	if !strings.Contains(out, "package.json") || !strings.Contains(out, "1.5.0 → 1.6.0") {
		t.Errorf("missing a version-file line:\n%s", out)
	}
}

func TestPlanViewWithoutVersionFiles(t *testing.T) {
	p := release.Plan{
		Current: semver.Version{Major: 1, Minor: 5, Prefix: "v"},
		Next:    semver.Version{Major: 1, Minor: 6, Prefix: "v"},
	}
	if strings.Contains(PlanView(p), "Version files") {
		t.Error("PlanView showed a Version files section with no changes")
	}
}

func TestPlanViewMarksPrerelease(t *testing.T) {
	p := release.Plan{
		Current:    semver.Version{Major: 1, Minor: 5, Prefix: "v"},
		Next:       semver.Version{Major: 1, Minor: 6, Prefix: "v", Pre: "rc.1"},
		Bump:       semver.Minor,
		Prerelease: true,
	}
	out := PlanView(p)
	if !strings.Contains(out, "v1.6.0-rc.1") || !strings.Contains(out, "(pre-release)") {
		t.Errorf("missing pre-release marker:\n%s", out)
	}
}

func TestPlanViewMarksFinalize(t *testing.T) {
	p := release.Plan{
		Current:    semver.Version{Major: 1, Minor: 6, Prefix: "v", Pre: "rc.2"},
		Next:       semver.Version{Major: 1, Minor: 6, Prefix: "v"},
		Bump:       semver.Minor,
		Finalizing: true,
	}
	out := PlanView(p)
	if !strings.Contains(out, "(finalize)") {
		t.Errorf("missing finalize marker:\n%s", out)
	}
}

func TestPlanViewListsHooks(t *testing.T) {
	p := release.Plan{
		Current: semver.Version{Major: 1, Minor: 5, Prefix: "v"},
		Next:    semver.Version{Major: 1, Minor: 6, Prefix: "v"},
	}
	p.Config.Release.Hooks = config.HooksConfig{
		Before: config.StringList{"make test"},
		After:  config.StringList{"./scripts/notify.sh", "echo done"},
	}
	out := PlanView(p)
	if !strings.Contains(out, "hooks") {
		t.Errorf("missing hooks line:\n%s", out)
	}
	if !strings.Contains(out, "before: make test") {
		t.Errorf("missing before hook:\n%s", out)
	}
	if !strings.Contains(out, "after: 2 commands") {
		t.Errorf("missing after hook count:\n%s", out)
	}
}

func TestPlanViewWithoutHooks(t *testing.T) {
	p := release.Plan{
		Current: semver.Version{Major: 1, Minor: 5, Prefix: "v"},
		Next:    semver.Version{Major: 1, Minor: 6, Prefix: "v"},
	}
	if strings.Contains(PlanView(p), "hooks") {
		t.Error("PlanView showed a hooks line with no hooks configured")
	}
}

// TestInfoAndSuccessMarkersAreLanguageInvariant is the PR4c-1 golden test for
// ui.go's warn/prompt/status-message wrapper functions. Info and Success each
// prepend a leading marker glyph ("· ", "✓") to a fully caller-supplied
// message — the same shape as pick.go's untouched "→ "+Label line (PR4a):
// the wrapper's own literal carries no translatable word, only a symbol, so
// there is nothing for i18n.T to localize. This test locks that finding in:
// a future change that accidentally routes either marker through i18n.T with
// per-language variation must fail here.
func TestInfoAndSuccessMarkersAreLanguageInvariant(t *testing.T) {
	t.Cleanup(func() { i18n.SetLanguage("en") })

	langs := i18n.Languages()
	if len(langs) < 2 {
		t.Fatalf("language registry has fewer than 2 entries (%d); this test proves nothing", len(langs))
	}

	if _, ok := i18n.SetLanguage("en"); !ok {
		t.Fatal("SetLanguage(en) rejected the reference language")
	}
	baselineInfo := Info("message")
	baselineSuccess := Success([]string{"done"})

	for _, lang := range langs {
		if _, ok := i18n.SetLanguage(lang.ID); !ok {
			t.Fatalf("SetLanguage(%q) rejected a registered language", lang.ID)
		}
		if got := Info("message"); got != baselineInfo {
			t.Errorf("Info diverged under language %q: got %q, want %q — Info's own literal is a glyph marker, not translatable text; only the caller-supplied message may vary", lang.ID, got, baselineInfo)
		}
		if got := Success([]string{"done"}); got != baselineSuccess {
			t.Errorf("Success diverged under language %q: got %q, want %q — Success's own literal is a glyph marker, not translatable text; only caller-supplied lines may vary", lang.ID, got, baselineSuccess)
		}
	}
}

func TestNotesShowsCommitHash(t *testing.T) {
	n := changelog.Notes{Groups: map[changelog.Group][]changelog.Item{
		changelog.Fixed: {
			{Text: "Crash on empty list", Hash: "abc1234"},
			{Text: "Header alignment", Hash: ""},
		},
	}}
	out := Notes(n)

	if !strings.Contains(out, "Crash on empty list") || !strings.Contains(out, "abc1234") {
		t.Errorf("expected fix text and hash in:\n%s", out)
	}
	// an item without a hash should not gain a stray marker
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "Header alignment") && strings.HasSuffix(strings.TrimSpace(line), "  ") {
			t.Errorf("trailing gap on hash-less line: %q", line)
		}
	}
}
