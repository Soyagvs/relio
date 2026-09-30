package releases

import (
	"os"

	"github.com/soyagvs/relio/internal/changelog"
)

// TagDeleter is the minimal repository capability Delete needs. It is
// satisfied structurally by *gitrepo.Repo (used by `relio releases delete`)
// and by repoPort's test fake (used by the interactive browser's tests) —
// no explicit implements declaration required.
type TagDeleter interface {
	DeleteTag(name string) error
}

// ChangelogError wraps a failure reading or writing the changelog after the
// git tag was already deleted, so callers can render "tag deleted, but the
// changelog step failed" differently from a tag-deletion failure (where
// nothing changed at all) — and, critically, know that the tag is already
// gone and any cached list of tags must be refreshed even though this call
// returned an error.
type ChangelogError struct{ Err error }

func (e *ChangelogError) Error() string { return e.Err.Error() }
func (e *ChangelogError) Unwrap() error { return e.Err }

// Delete deletes the given tag and, when changelogPath has a matching
// "## [version]" section, removes that section too. It is the single
// implementation of "delete this release" shared by the interactive browser
// (doDelete) and `relio releases delete` — the two callers must not
// reimplement these steps independently.
//
// changelogRemoved reports whether a changelog section was found and
// removed. A missing changelog file, or one with no section for version, is
// not an error — only the tag is required to exist.
//
// If the tag deletes successfully but RemoveChangelogSection then fails, the
// tag is already gone and cannot be "rolled back" (Delete does not attempt
// to recreate it) — callers must be able to resume just the changelog step
// on a retry, since resolving version back to a tag name will no longer
// work once the tag is gone. See HasStaleChangelogSection.
func Delete(repo TagDeleter, changelogPath, version string) (changelogRemoved bool, err error) {
	if err := repo.DeleteTag(version); err != nil {
		return false, err
	}
	return RemoveChangelogSection(changelogPath, version)
}

// RemoveChangelogSection removes changelogPath's "## [version]" section, if
// one exists. It never touches git — callers that also need the tag deleted
// use Delete, which calls this as its second step. Exposed separately so a
// caller can resume just this step once the tag is already gone (e.g. a
// retry after Delete's tag-deletion step succeeded but this one failed the
// first time): once the tag no longer exists, there is no tag name left to
// re-resolve, so the changelog cleanup must be reachable on its own.
//
// removed reports whether a section was found and removed. A missing
// changelog file, or one with no section for version, is not an error.
// A failure here is wrapped in *ChangelogError so callers can distinguish it
// from an error that means nothing happened at all.
func RemoveChangelogSection(changelogPath, version string) (removed bool, err error) {
	data, err := os.ReadFile(changelogPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, &ChangelogError{Err: err}
	}

	content := string(data)
	if changelog.ExtractSection(content, version) == "" {
		return false, nil
	}

	updated := changelog.RemoveSection(content, version)
	if err := os.WriteFile(changelogPath, []byte(updated), 0o644); err != nil {
		return false, &ChangelogError{Err: err}
	}
	return true, nil
}

// HasStaleChangelogSection reports whether changelogPath still has a
// "## [version]" section for version, with no requirement that a matching
// git tag exists. `relio releases delete` uses this when version no longer
// resolves to a tag, to tell "already fully deleted, nothing to do" apart
// from "the tag is gone but a previous delete's changelog step failed,
// there is still a stale section to clean up."
func HasStaleChangelogSection(changelogPath, version string) bool {
	data, err := os.ReadFile(changelogPath)
	if err != nil {
		return false
	}
	return changelog.ExtractSection(string(data), version) != ""
}
