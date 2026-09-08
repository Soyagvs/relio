// Package versionfile keeps declared project files in sync with the release
// version. It plans regex-based edits — no format-aware parsing — and applies
// them in place, preserving the file's surrounding bytes and permissions so the
// edit rides in the same commit as the changelog.
package versionfile

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Target is one declared file to keep in sync. Pattern is a Go regexp with
// exactly one capture group wrapping the version substring; empty Pattern means
// "use the built-in rule for this filename".
type Target struct {
	Path    string
	Pattern string
}

// Change is a pending edit: replace Old with New at the matched span in Path.
// Pattern carries the target's pattern (empty for a built-in rule) so Apply can
// re-locate the exact span with the same regex instead of a plain text search —
// that keeps it from editing an identical version string elsewhere in the file.
type Change struct {
	Path    string // absolute
	Rel     string // path as declared, for display
	Pattern string
	Old     string
	New     string
}

// Plan resolves every target against root, locates the version span in each, and
// returns the edits to make. It never writes. version is the bare number with no
// leading "v". Changes come back in declared order; an idempotent re-run still
// returns a Change with Old == New.
func Plan(root string, targets []Target, version string) ([]Change, error) {
	var changes []Change
	for _, t := range targets {
		path := t.Path
		if !filepath.IsAbs(path) {
			path = filepath.Join(root, path)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				return nil, fmt.Errorf("versionfile: %s: file not found", t.Path)
			}
			return nil, fmt.Errorf("versionfile: %s: %w", t.Path, err)
		}
		re, group, err := ruleFor(t.Path, t.Pattern)
		if err != nil {
			return nil, err
		}
		loc := re.FindSubmatchIndex(data)
		if loc == nil || 2*group+1 >= len(loc) || loc[2*group] < 0 {
			return nil, fmt.Errorf("versionfile: %s: no version match (add an explicit pattern?)", t.Path)
		}
		old := string(data[loc[2*group]:loc[2*group+1]])
		changes = append(changes, Change{Path: path, Rel: t.Path, Pattern: t.Pattern, Old: old, New: version})
	}
	return changes, nil
}

// Apply performs the planned edits. For each Change it re-reads the file,
// re-locates the version span with the same rule Plan used, and swaps it for
// New — never a plain text search, so an identical version string elsewhere in
// the file is left alone. It writes back with the file's existing permission
// (0o644 when it cannot be read) and stops at the first error.
func Apply(changes []Change) error {
	for _, c := range changes {
		if c.Old == c.New {
			continue
		}
		data, err := os.ReadFile(c.Path)
		if err != nil {
			return fmt.Errorf("versionfile: %s: %w", c.Rel, err)
		}
		re, group, err := ruleFor(c.Rel, c.Pattern)
		if err != nil {
			return err
		}
		loc := re.FindSubmatchIndex(data)
		if loc == nil || 2*group+1 >= len(loc) || loc[2*group] < 0 {
			return fmt.Errorf("versionfile: %s: version span not found (did the file change?)", c.Rel)
		}
		start, end := loc[2*group], loc[2*group+1]
		if string(data[start:end]) != c.Old {
			return fmt.Errorf("versionfile: %s: file changed since planning (found %q, expected %q)",
				c.Rel, string(data[start:end]), c.Old)
		}
		updated := string(data[:start]) + c.New + string(data[end:])

		perm := os.FileMode(0o644)
		if info, serr := os.Stat(c.Path); serr == nil {
			perm = info.Mode().Perm()
		}
		if err := os.WriteFile(c.Path, []byte(updated), perm); err != nil {
			return fmt.Errorf("versionfile: %s: %w", c.Rel, err)
		}
	}
	return nil
}

// ruleFor returns the regexp to locate the version and the index of the capture
// group that wraps it. An explicit pattern must have exactly one group.
func ruleFor(rel, pattern string) (re *regexp.Regexp, group int, err error) {
	if pattern != "" {
		re, err = regexp.Compile(pattern)
		if err != nil {
			return nil, 0, fmt.Errorf("versionfile: %s: %w", rel, err)
		}
		if re.NumSubexp() != 1 {
			return nil, 0, fmt.Errorf("versionfile: %s: pattern needs exactly one capture group, has %d", rel, re.NumSubexp())
		}
		return re, 1, nil
	}

	base := filepath.Base(rel)
	switch {
	case base == "package.json":
		return regexp.MustCompile(`("version"\s*:\s*")([^"]+)(")`), 2, nil
	case base == "Cargo.toml", base == "pyproject.toml", strings.HasSuffix(base, ".toml"):
		return regexp.MustCompile(`(?m)^(version\s*=\s*")([^"]+)(")`), 2, nil
	case base == "VERSION", base == "version.txt":
		return regexp.MustCompile(`\s*(\S+)\s*`), 1, nil
	default:
		return nil, 0, fmt.Errorf("versionfile: %s: no built-in rule, set a pattern", rel)
	}
}
