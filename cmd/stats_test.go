package cmd

import (
	"strings"
	"testing"

	"github.com/soyagvs/relio/internal/ghstats"
	"github.com/soyagvs/relio/internal/i18n"
)

func TestNum(t *testing.T) {
	cases := map[int]string{0: "0", 42: "42", 1284: "1,284", 1000000: "1,000,000", -5000: "-5,000"}
	for in, want := range cases {
		if got := num(in); got != want {
			t.Errorf("num(%d) = %q, want %q", in, got, want)
		}
	}
}

func TestRenderStats(t *testing.T) {
	latest := ghstats.Release{
		Tag:       "v1.2.0",
		Downloads: 437,
		Assets: []ghstats.Asset{
			{Name: "relio_1.2.0_darwin_arm64.tar.gz", DownloadCount: 291, Platform: "macOS arm64"},
			{Name: "relio_1.2.0_darwin_amd64.tar.gz", DownloadCount: 38, Platform: "macOS amd64"},
			{Name: "relio_1.2.0_linux_amd64.tar.gz", DownloadCount: 82, Platform: "Linux amd64"},
			{Name: "relio_1.2.0_linux_arm64.tar.gz", DownloadCount: 26, Platform: "Linux arm64"},
			{Name: "checksums.txt", DownloadCount: 5},
		},
	}
	s := &ghstats.Stats{
		Repo:           "soyagvs/relio",
		Stars:          126,
		Forks:          14,
		TotalDownloads: 1284,
		Releases: []ghstats.Release{
			latest,
			{Tag: "v1.1.0", Downloads: 521},
			{Tag: "v1.0.0", Downloads: 326},
		},
	}
	s.Latest = &s.Releases[0]

	out := renderStats(s, false)
	for _, want := range []string{
		"Relio -- stats",
		"total", "1,284",
		"latest release", "437",
		"v1.2.0", "521", "326",
		"macOS arm64", "291",
		"Linux arm64", "26",
		"stars", "126",
		"forks", "14",
		"downloads = release-asset downloads",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("render missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "checksums") && strings.Contains(out, "checksums.txt   ") {
		t.Errorf("checksums.txt should not appear as a platform row:\n%s", out)
	}
}

func TestRenderStatsHidesPrereleasesByDefault(t *testing.T) {
	s := &ghstats.Stats{
		Repo: "x/y",
		Releases: []ghstats.Release{
			{Tag: "v1.0.0", Downloads: 10},
			{Tag: "v1.1.0-rc.1", Downloads: 3, Prerelease: true},
		},
	}
	s.Latest = &s.Releases[0]

	if strings.Contains(renderStats(s, false), "rc.1") {
		t.Error("prerelease shown without --prerelease")
	}
	if !strings.Contains(renderStats(s, true), "rc.1") {
		t.Error("prerelease hidden with --prerelease")
	}
}

// TestNewStatsCmdShortLongLocalizeAtConstructionTime proves newStatsCmd()'s
// Short/Long and --repo/--prerelease flag usage strings resolve through the
// active i18n catalog at construction time, matching the ordering proof
// established for cmd/root.go in PR6a.
func TestNewStatsCmdShortLongLocalizeAtConstructionTime(t *testing.T) {
	prev := i18n.Current()
	t.Cleanup(func() { i18n.SetLanguage(prev) })

	i18n.SetLanguage("en")
	cmdEN := newStatsCmd()
	i18n.SetLanguage("es")
	cmdES := newStatsCmd()

	if cmdES.Short == cmdEN.Short {
		t.Errorf("stats Short unchanged across languages: %q", cmdES.Short)
	}
	if cmdES.Long == cmdEN.Long {
		t.Error("stats Long unchanged across languages")
	}
	for _, name := range []string{"repo", "prerelease"} {
		fEN := cmdEN.Flags().Lookup(name)
		fES := cmdES.Flags().Lookup(name)
		if fEN == nil || fES == nil {
			t.Fatalf("flag %q not found", name)
		}
		if fEN.Usage == fES.Usage {
			t.Errorf("--%s usage unchanged across languages: %q", name, fES.Usage)
		}
	}
}

// TestGoldenEnglishStatsCmdUnchanged pins the hardcoded English literals in
// cmd/stats.go against i18n.T() under the default "en" language: converting
// them to i18n.T() calls MUST NOT change a single byte of English output.
func TestGoldenEnglishStatsCmdUnchanged(t *testing.T) {
	prev := i18n.Current()
	if prev != "en" {
		i18n.SetLanguage("en")
	}
	t.Cleanup(func() { i18n.SetLanguage(prev) })

	c := newStatsCmd()
	if c.Short != "Show Relio's public download and GitHub stats" {
		t.Errorf("Short = %q", c.Short)
	}
	wantLong := "Read-only public statistics from the GitHub REST API: release asset\n" +
		"download counts, per-release and per-platform breakdowns, stars and forks.\n\n" +
		"\"Downloads\" = times a release asset was downloaded — NOT unique users or\n" +
		"active installs. No authentication is required; set GITHUB_TOKEN to raise\n" +
		"the API rate limit. Relio sends no telemetry of any kind."
	if c.Long != wantLong {
		t.Errorf("Long = %q, want %q", c.Long, wantLong)
	}
	if f := c.Flags().Lookup("repo"); f == nil || f.Usage != "owner/name to query" {
		t.Errorf("--repo usage = %q", f.Usage)
	}
	if f := c.Flags().Lookup("prerelease"); f == nil || f.Usage != "include pre-releases in the release list" {
		t.Errorf("--prerelease usage = %q", f.Usage)
	}
}

// TestRenderStatsLocalizesLabels proves renderStats' section headers, row
// labels, and footer note resolve through the active catalog.
func TestRenderStatsLocalizesLabels(t *testing.T) {
	prev := i18n.Current()
	t.Cleanup(func() { i18n.SetLanguage(prev) })

	s := &ghstats.Stats{
		Repo: "x/y",
		Releases: []ghstats.Release{
			{Tag: "v1.0.0", Downloads: 10},
		},
	}
	s.Latest = &s.Releases[0]

	i18n.SetLanguage("en")
	outEN := renderStats(s, false)
	i18n.SetLanguage("es")
	outES := renderStats(s, false)

	if outEN == outES {
		t.Error("renderStats output unchanged across languages")
	}
	if !strings.Contains(outEN, "Downloads") || !strings.Contains(outES, "Descargas") {
		t.Errorf("Downloads section not localized: en=%q es=%q", outEN, outES)
	}
	if !strings.Contains(outEN, "GitHub") || !strings.Contains(outES, "GitHub") {
		t.Errorf("GitHub section missing: en=%q es=%q", outEN, outES)
	}
	if !strings.Contains(outEN, "downloads = release-asset downloads") {
		t.Errorf("footer note missing from en output: %q", outEN)
	}
}
