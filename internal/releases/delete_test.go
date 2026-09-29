package releases

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// deleteFixtureChangelog mirrors the shape releases.go's own fixtures use.
const deleteFixtureChangelog = "# Changelog\n\n" +
	"## [0.2.0] - 2026-09-06\n\n### Added\n\n- Second thing\n\n" +
	"## [0.1.0] - 2026-08-01\n\n### Added\n\n- First thing\n"

func writeDeleteFixture(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "CHANGELOG.md")
	if err := os.WriteFile(path, []byte(deleteFixtureChangelog), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestDeleteRemovesTagAndChangelogSection is the shared implementation's
// happy path — the same two steps doDelete already exercises via the TUI.
func TestDeleteRemovesTagAndChangelogSection(t *testing.T) {
	dir := t.TempDir()
	path := writeDeleteFixture(t, dir)

	repo := &fakeRepo{}
	changelogRemoved, err := Delete(repo, path, "v0.2.0")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if !changelogRemoved {
		t.Error("changelogRemoved = false, want true")
	}
	if len(repo.deleted) != 1 || repo.deleted[0] != "v0.2.0" {
		t.Errorf("deleted = %v, want [v0.2.0]", repo.deleted)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "## [0.2.0]") || strings.Contains(string(data), "Second thing") {
		t.Errorf("changelog section not removed:\n%s", data)
	}
	if !strings.Contains(string(data), "## [0.1.0]") {
		t.Errorf("wrong section removed:\n%s", data)
	}
}

// TestDeleteWithoutMatchingChangelogSection proves the tag is still deleted
// even when the changelog has no section for it, and reports
// changelogRemoved=false rather than erroring.
func TestDeleteWithoutMatchingChangelogSection(t *testing.T) {
	dir := t.TempDir()
	path := writeDeleteFixture(t, dir)

	repo := &fakeRepo{}
	changelogRemoved, err := Delete(repo, path, "v9.9.9")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if changelogRemoved {
		t.Error("changelogRemoved = true, want false for a version absent from the changelog")
	}
	if len(repo.deleted) != 1 || repo.deleted[0] != "v9.9.9" {
		t.Errorf("deleted = %v, want [v9.9.9]", repo.deleted)
	}
}

// TestDeleteMissingChangelogFileIsNotAnError proves a missing changelog file
// (e.g. a project without one) does not fail the delete — only the tag is
// required to exist.
func TestDeleteMissingChangelogFileIsNotAnError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "CHANGELOG.md") // never written

	repo := &fakeRepo{}
	changelogRemoved, err := Delete(repo, path, "v1.0.0")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if changelogRemoved {
		t.Error("changelogRemoved = true, want false when no changelog file exists")
	}
}

// TestDeleteTagFailurePropagatesRawError proves a tag-deletion failure is
// returned as-is (not wrapped) and never touches the changelog file.
func TestDeleteTagFailurePropagatesRawError(t *testing.T) {
	dir := t.TempDir()
	path := writeDeleteFixture(t, dir)

	wantErr := errors.New("git tag -d: boom")
	repo := &fakeRepo{deleteErr: wantErr}
	_, err := Delete(repo, path, "v0.2.0")
	if !errors.Is(err, wantErr) {
		t.Fatalf("Delete err = %v, want %v", err, wantErr)
	}

	data, rerr := os.ReadFile(path)
	if rerr != nil {
		t.Fatal(rerr)
	}
	if string(data) != deleteFixtureChangelog {
		t.Errorf("changelog modified despite tag-delete failure:\n%s", data)
	}
}

// TestDeleteReadErrorPropagatesRaw proves a changelog read failure that is
// NOT "file does not exist" (e.g. the path is a directory) is returned
// as-is, distinguishing it from the "no changelog file" case above.
func TestDeleteReadErrorPropagatesRaw(t *testing.T) {
	dir := t.TempDir()
	// A directory at the changelog path makes os.ReadFile fail with an error
	// that is not os.IsNotExist.
	asDir := filepath.Join(dir, "CHANGELOG.md")
	if err := os.Mkdir(asDir, 0o755); err != nil {
		t.Fatal(err)
	}

	repo := &fakeRepo{}
	_, err := Delete(repo, asDir, "v0.2.0")
	if err == nil {
		t.Fatal("Delete err = nil, want non-nil when the changelog path is a directory")
	}
	var cwErr *ChangelogWriteError
	if errors.As(err, &cwErr) {
		t.Errorf("Delete err should not be a ChangelogWriteError for a read failure, got %v", err)
	}
}

// TestDeleteChangelogWriteFailureWrapsError proves a failure writing the
// updated changelog — after the tag was already deleted — is wrapped in
// ChangelogWriteError so callers can render it distinctly from a
// tag-deletion failure.
func TestDeleteChangelogWriteFailureWrapsError(t *testing.T) {
	dir := t.TempDir()
	path := writeDeleteFixture(t, dir)
	if err := os.Chmod(path, 0o400); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(path, 0o644) })

	repo := &fakeRepo{}
	changelogRemoved, err := Delete(repo, path, "v0.2.0")
	if err == nil {
		t.Fatal("Delete err = nil, want non-nil when the changelog file is not writable")
	}
	if changelogRemoved {
		t.Error("changelogRemoved = true, want false when the write failed")
	}
	var cwErr *ChangelogWriteError
	if !errors.As(err, &cwErr) {
		t.Fatalf("Delete err = %v (%T), want a *ChangelogWriteError", err, err)
	}
	// The tag deletion itself must still have gone through.
	if len(repo.deleted) != 1 || repo.deleted[0] != "v0.2.0" {
		t.Errorf("deleted = %v, want [v0.2.0] even though the changelog write failed", repo.deleted)
	}
}
