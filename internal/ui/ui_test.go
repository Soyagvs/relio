package ui

import (
	"bytes"
	"regexp"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/soyagvs/relio/internal/changelog"
	"github.com/soyagvs/relio/internal/config"
	"github.com/soyagvs/relio/internal/release"
	"github.com/soyagvs/relio/internal/semver"
	"github.com/soyagvs/relio/internal/versionfile"
)

var ansiRE = regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]`)

func stripANSI(s string) string { return ansiRE.ReplaceAllString(s, "") }

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
