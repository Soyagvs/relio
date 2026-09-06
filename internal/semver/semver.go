// Package semver implements the minimal subset of Semantic Versioning the
// release tool needs: parsing MAJOR.MINOR.PATCH, bumping it, and deciding which
// bump a set of Conventional Commits requires.
package semver

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/soyagvs/go-release/internal/conventional"
)

// Bump is the kind of version increment.
type Bump int

const (
	None Bump = iota
	Patch
	Minor
	Major
)

func (b Bump) String() string {
	switch b {
	case Patch:
		return "patch"
	case Minor:
		return "minor"
	case Major:
		return "major"
	default:
		return "none"
	}
}

// Version is a parsed MAJOR.MINOR.PATCH triple. Prefix preserves a leading "v".
type Version struct {
	Major, Minor, Patch int
	Prefix              string
}

// Zero is the starting point for a repository with no tags.
func Zero() Version { return Version{0, 0, 0, "v"} }

// Parse reads "v1.2.3" or "1.2.3". Pre-release and build metadata are rejected
// for the MVP so the tool never guesses at ambiguous inputs.
func Parse(s string) (Version, error) {
	raw := strings.TrimSpace(s)
	v := Version{}
	if strings.HasPrefix(raw, "v") || strings.HasPrefix(raw, "V") {
		v.Prefix = "v"
		raw = raw[1:]
	}
	if strings.ContainsAny(raw, "-+") {
		return Version{}, fmt.Errorf("semver: pre-release/build metadata not supported: %q", s)
	}
	parts := strings.Split(raw, ".")
	if len(parts) != 3 {
		return Version{}, fmt.Errorf("semver: want MAJOR.MINOR.PATCH, got %q", s)
	}
	nums := [3]int{}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return Version{}, fmt.Errorf("semver: invalid number %q in %q", p, s)
		}
		nums[i] = n
	}
	v.Major, v.Minor, v.Patch = nums[0], nums[1], nums[2]
	return v, nil
}

// String renders the version, keeping any original "v" prefix.
func (v Version) String() string {
	return fmt.Sprintf("%s%d.%d.%d", v.Prefix, v.Major, v.Minor, v.Patch)
}

// Next returns the version after applying b. None returns v unchanged.
func (v Version) Next(b Bump) Version {
	switch b {
	case Major:
		return Version{v.Major + 1, 0, 0, v.Prefix}
	case Minor:
		return Version{v.Major, v.Minor + 1, 0, v.Prefix}
	case Patch:
		return Version{v.Major, v.Minor, v.Patch + 1, v.Prefix}
	default:
		return v
	}
}

// BumpFor decides the required bump from a set of commits:
//
//	any breaking change      -> Major
//	otherwise any feat       -> Minor
//	otherwise any fix/perf/refactor -> Patch
//	otherwise                -> None
func BumpFor(commits []conventional.Commit) Bump {
	result := None
	for _, c := range commits {
		if c.Breaking {
			return Major
		}
		switch c.Type {
		case "feat":
			if result < Minor {
				result = Minor
			}
		case "fix", "perf", "refactor":
			if result < Patch {
				result = Patch
			}
		}
	}
	return result
}
