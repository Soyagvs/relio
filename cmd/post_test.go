package cmd

import (
	"strings"
	"testing"
	"time"

	"github.com/soyagvs/relio/internal/changelog"
	"github.com/soyagvs/relio/internal/conventional"
	"github.com/soyagvs/relio/internal/release"
	"github.com/soyagvs/relio/internal/semver"
	"github.com/soyagvs/relio/internal/ui"
)

func samplePlan() release.Plan {
	commits := []conventional.Commit{
		{Type: "feat", Description: "add transaction categories", Hash: "1111111111111111111111111111111111111111"},
		{Type: "feat", Scope: "dashboard", Description: "monthly summary widget", Hash: "2222222222222222222222222222222222222222"},
		{Type: "fix", Description: "crash on empty account list", Hash: "3333333333333333333333333333333333333333"},
		{Type: "fix", Scope: "kiosk", Description: "header alignment", Hash: "4444444444444444444444444444444444444444"},
		{Type: "refactor", Description: "auth flow", Hash: "5555555555555555555555555555555555555555"},
		{Type: "chore", Description: "bump deps", Hash: "6666666666666666666666666666666666666666"},
	}
	return release.Plan{
		Current: semver.Version{Major: 1, Minor: 3, Patch: 2, Prefix: "v"},
		Next:    semver.Version{Major: 1, Minor: 4, Patch: 0, Prefix: "v"},
		Bump:    semver.Minor,
		Commits: commits,
		Notes:   changelog.Build(commits),
		Now:     time.Date(2026, 9, 6, 14, 30, 0, 0, time.UTC),
	}
}

func TestMinimalPost(t *testing.T) {
	got := minimalPost("gestam-frontend", samplePlan())

	for _, w := range []string{
		"relio -- release",
		"gestam-frontend · v1.4.0",
		"2026-09-06 · 14:30 · 6 commits",
		"Added",
		"1111111  Add transaction categories",
		"Fixed",
		"3333333  Crash on empty account list",
		"4444444  kiosk: Header alignment",
	} {
		if !strings.Contains(got, w) {
			t.Errorf("missing %q in:\n%s", w, got)
		}
	}
	if strings.Contains(got, "v1.3.2..") {
		t.Errorf("commit range should be gone:\n%s", got)
	}

	lines := strings.Split(got, "\n")
	if lines[0] != "relio -- release" {
		t.Errorf("line 1 = %q", lines[0])
	}
	if lines[2] != "gestam-frontend · v1.4.0" {
		t.Errorf("line 3 = %q", lines[2])
	}
	if lines[3] != "2026-09-06 · 14:30 · 6 commits" {
		t.Errorf("line 4 = %q", lines[3])
	}
}

func TestSocialPost(t *testing.T) {
	got := socialPost("azeink", samplePlan())
	lines := strings.Split(got, "\n")

	if lines[0] != "azeink -- Release" && lines[0] != "Azeink -- Release" {
		t.Errorf("line 1 = %q", lines[0])
	}
	if lines[2] != "v1.4.0 · 06.09.26 · 14:30" {
		t.Errorf("meta line = %q", lines[2])
	}
	hasRow := func(typ, desc string) bool {
		for _, ln := range lines {
			if strings.HasPrefix(strings.TrimSpace(ln), typ) && strings.Contains(ln, desc) {
				return true
			}
		}
		return false
	}
	if !hasRow("feat", "Add transaction categories") || !hasRow("fix", "Crash on empty account list") {
		t.Errorf("expected feat/fix rows in:\n%s", got)
	}
	// chore is dropped, hashes never shown
	if strings.Contains(got, "bump deps") || strings.Contains(got, "1111111") {
		t.Errorf("social post should be filtered and hash-free:\n%s", got)
	}
}

func TestNoHashHidesHashes(t *testing.T) {
	ui.HideHashes = true
	defer func() { ui.HideHashes = false }()

	got := minimalPost("proj", samplePlan())
	if strings.Contains(got, "1111111") || strings.Contains(got, "3333333") {
		t.Errorf("--no-hash should drop hashes:\n%s", got)
	}
	if !strings.Contains(got, "• Add transaction categories") {
		t.Errorf("expected plain bullet lines:\n%s", got)
	}
}

// minimalPost must be byte-for-byte the releases-browser output for the same
// project, version, meta and notes.
func TestMinimalPostMatchesReleasesBrowser(t *testing.T) {
	p := samplePlan()
	got := minimalPost("gestam-frontend", p)
	want := ui.ReleaseText("gestam-frontend", p.Next.String(), commitMeta(p), p.Notes)
	if got != want {
		t.Errorf("minimalPost diverged from ui.ReleaseText:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestMinimalPostNoNotableChanges(t *testing.T) {
	commits := []conventional.Commit{{Type: "chore", Description: "bump deps"}}
	p := release.Plan{
		Next:    semver.Version{Minor: 1, Prefix: "v"},
		Commits: commits,
		Notes:   changelog.Build(commits),
		Now:     time.Date(2026, 9, 6, 14, 30, 0, 0, time.UTC),
	}
	got := minimalPost("proj", p)
	if !strings.Contains(got, "1 commits") {
		t.Errorf("commit count missing:\n%s", got)
	}
	if !strings.Contains(got, "(no user-facing changes)") {
		t.Errorf("expected the no-changes note:\n%s", got)
	}
}
