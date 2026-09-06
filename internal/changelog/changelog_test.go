package changelog

import (
	"strings"
	"testing"
	"time"

	"github.com/soyagvs/go-release/internal/conventional"
)

var fixedDate = time.Date(2026, 9, 6, 0, 0, 0, 0, time.UTC)

func sampleCommits() []conventional.Commit {
	return []conventional.Commit{
		{Type: "feat", Description: "add facial attendance"},
		{Type: "fix", Scope: "kiosk", Description: "header alignment"},
		{Type: "refactor", Description: "auth flow"},
		{Type: "chore", Description: "bump deps"},
	}
}

func TestBuildGrouping(t *testing.T) {
	n := Build(sampleCommits())
	if got := n.Groups[Added]; len(got) != 1 || got[0] != "Add facial attendance" {
		t.Errorf("Added = %v", got)
	}
	if got := n.Groups[Fixed]; len(got) != 1 || got[0] != "kiosk: Header alignment" {
		t.Errorf("Fixed = %v", got)
	}
	if got := n.Groups[Changed]; len(got) != 1 || got[0] != "Auth flow" {
		t.Errorf("Changed = %v", got)
	}
	if n.Empty() {
		t.Error("notes should not be empty")
	}
}

func TestBuildBreaking(t *testing.T) {
	n := Build([]conventional.Commit{{Type: "feat", Description: "new api", Breaking: true}})
	if got := n.Groups[Changed]; len(got) != 1 || !strings.HasPrefix(got[0], "**Breaking:**") {
		t.Errorf("Changed = %v", got)
	}
}

func TestRenderSection(t *testing.T) {
	got := RenderSection("v1.4.0", fixedDate, Build(sampleCommits()))
	want := `## [1.4.0] - 2026-09-06

### Added

- Add facial attendance

### Changed

- Auth flow

### Fixed

- kiosk: Header alignment`
	if got != want {
		t.Errorf("RenderSection mismatch:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestRenderSectionNoNotableChanges(t *testing.T) {
	got := RenderSection("v1.0.1", fixedDate, Build([]conventional.Commit{{Type: "chore", Description: "x"}}))
	if !strings.Contains(got, "_No user-facing changes._") {
		t.Errorf("expected placeholder, got:\n%s", got)
	}
}

func TestUpdateSeedsEmptyFile(t *testing.T) {
	section := RenderSection("v0.1.0", fixedDate, Build(sampleCommits()))
	got := Update("", section)
	if !strings.HasPrefix(got, "# Changelog") {
		t.Errorf("expected header, got:\n%s", got)
	}
	if !strings.Contains(got, "## [0.1.0] - 2026-09-06") {
		t.Errorf("expected section, got:\n%s", got)
	}
	if !strings.HasSuffix(got, "\n") || strings.HasSuffix(got, "\n\n") {
		t.Errorf("expected single trailing newline")
	}
}

func TestUpdateInsertsBeforeExistingReleases(t *testing.T) {
	existing := Header + "\n## [1.3.2] - 2026-08-01\n\n### Fixed\n\n- Old bug\n"
	section := RenderSection("v1.4.0", fixedDate, Build(sampleCommits()))
	got := Update(existing, section)

	iNew := strings.Index(got, "## [1.4.0]")
	iOld := strings.Index(got, "## [1.3.2]")
	if iNew < 0 || iOld < 0 {
		t.Fatalf("missing sections:\n%s", got)
	}
	if iNew > iOld {
		t.Errorf("new release must come before old one:\n%s", got)
	}
	if !strings.Contains(got, "- Old bug") {
		t.Errorf("old content lost:\n%s", got)
	}
}
