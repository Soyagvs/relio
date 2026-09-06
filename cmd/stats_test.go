package cmd

import (
	"strings"
	"testing"

	"github.com/soyagvs/relio/internal/ghstats"
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
