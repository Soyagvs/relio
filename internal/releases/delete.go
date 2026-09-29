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

// ChangelogWriteError wraps a failure writing the changelog after the git tag
// was already deleted, so callers can render "tag deleted, but the changelog
// write failed" differently from a tag-deletion failure (where nothing
// changed at all).
type ChangelogWriteError struct{ Err error }

func (e *ChangelogWriteError) Error() string { return e.Err.Error() }
func (e *ChangelogWriteError) Unwrap() error { return e.Err }

// Delete deletes the given tag and, when changelogPath has a matching
// "## [version]" section, removes that section too. It is the single
// implementation of "delete this release" shared by the interactive browser
// (doDelete) and `relio releases delete` — the two callers must not
// reimplement these steps independently.
//
// changelogRemoved reports whether a changelog section was found and
// removed. A missing changelog file, or one with no section for version, is
// not an error — only the tag is required to exist.
func Delete(repo TagDeleter, changelogPath, version string) (changelogRemoved bool, err error) {
	if err := repo.DeleteTag(version); err != nil {
		return false, err
	}

	data, err := os.ReadFile(changelogPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	content := string(data)
	if changelog.ExtractSection(content, version) == "" {
		return false, nil
	}

	updated := changelog.RemoveSection(content, version)
	if err := os.WriteFile(changelogPath, []byte(updated), 0o644); err != nil {
		return false, &ChangelogWriteError{Err: err}
	}
	return true, nil
}
