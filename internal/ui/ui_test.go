package ui

import (
	"strings"
	"testing"

	"github.com/soyagvs/relio/internal/changelog"
	"github.com/soyagvs/relio/internal/release"
	"github.com/soyagvs/relio/internal/semver"
	"github.com/soyagvs/relio/internal/versionfile"
)

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
