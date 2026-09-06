package cmd

import (
	"strings"
	"testing"
	"time"

	"github.com/soyagvs/go-release/internal/changelog"
	"github.com/soyagvs/go-release/internal/conventional"
	"github.com/soyagvs/go-release/internal/release"
	"github.com/soyagvs/go-release/internal/semver"
)

func samplePlan() release.Plan {
	commits := []conventional.Commit{
		{Type: "feat", Description: "add transaction categories"},
		{Type: "feat", Scope: "dashboard", Description: "monthly summary widget"},
		{Type: "fix", Description: "crash on empty account list"},
		{Type: "fix", Scope: "kiosk", Description: "header alignment"},
		{Type: "refactor", Description: "auth flow"},
		{Type: "chore", Description: "bump deps"},
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

	wantLines := []string{
		"--- gestam-frontend release ---",
		"v1.4.0",
		"2026-09-06 14:30",
		"6 commits",
		"Fixes",
		"- Crash on empty account list",
		"- kiosk: Header alignment",
	}
	for _, w := range wantLines {
		if !strings.Contains(got, w) {
			t.Errorf("missing %q in:\n%s", w, got)
		}
	}

	// title is the first line, version the second, timestamp the third
	lines := strings.Split(got, "\n")
	if lines[0] != "--- gestam-frontend release ---" {
		t.Errorf("line 1 = %q", lines[0])
	}
	if lines[1] != "v1.4.0" {
		t.Errorf("line 2 = %q", lines[1])
	}
	if lines[2] != "2026-09-06 14:30" {
		t.Errorf("line 3 = %q", lines[2])
	}
}

func TestMinimalPostMaintenanceOnly(t *testing.T) {
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
	if !strings.Contains(got, "Maintenance release.") {
		t.Errorf("expected maintenance fallback:\n%s", got)
	}
}
