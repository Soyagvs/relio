// Package semver implements the minimal subset of Semantic Versioning the
// release tool needs: parsing MAJOR.MINOR.PATCH (optionally with an "-rc.N"
// style pre-release), bumping it, and deciding which bump a set of Conventional
// Commits requires.
package semver

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/soyagvs/relio/internal/conventional"
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

// Version is a parsed MAJOR.MINOR.PATCH triple, optionally carrying a
// pre-release. Prefix preserves a leading "v". Pre holds the pre-release
// identifiers without the leading "-" (e.g. "rc.1"); an empty Pre means a
// stable release.
type Version struct {
	Major, Minor, Patch int
	Prefix              string
	Pre                 string
}

// Zero is the starting point for a repository with no tags.
func Zero() Version { return Version{Major: 0, Minor: 0, Patch: 0, Prefix: "v"} }

// Parse reads "v1.2.3" or "1.2.3", optionally with a pre-release
// ("v1.6.0-rc.1"). The string is split on the FIRST "-": the left part is the
// MAJOR.MINOR.PATCH core, the right part is stored verbatim in Pre. Build
// metadata ("+...") is still rejected, as is a trailing "-" and a pre-release
// containing anything outside ASCII letters, digits, ".", and "-".
func Parse(s string) (Version, error) {
	raw := strings.TrimSpace(s)
	v := Version{}
	if strings.HasPrefix(raw, "v") || strings.HasPrefix(raw, "V") {
		v.Prefix = "v"
		raw = raw[1:]
	}
	if strings.Contains(raw, "+") {
		return Version{}, fmt.Errorf("semver: build metadata not supported: %q", s)
	}
	core := raw
	if i := strings.IndexByte(raw, '-'); i >= 0 {
		core = raw[:i]
		pre := raw[i+1:]
		if pre == "" {
			return Version{}, fmt.Errorf("semver: empty pre-release in %q", s)
		}
		if !validPre(pre) {
			return Version{}, fmt.Errorf("semver: invalid pre-release %q in %q", pre, s)
		}
		v.Pre = pre
	}
	parts := strings.Split(core, ".")
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

// validPre reports whether pre is made up only of ASCII letters, digits, dots
// and hyphens.
func validPre(pre string) bool {
	for _, r := range pre {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '.', r == '-':
		default:
			return false
		}
	}
	return true
}

// String renders the version, keeping any original "v" prefix and appending
// "-<Pre>" for a pre-release.
func (v Version) String() string {
	s := fmt.Sprintf("%s%d.%d.%d", v.Prefix, v.Major, v.Minor, v.Patch)
	if v.Pre != "" {
		s += "-" + v.Pre
	}
	return s
}

// IsPrerelease reports whether v carries a pre-release identifier.
func (v Version) IsPrerelease() bool { return v.Pre != "" }

// Core returns a copy of v with the pre-release stripped.
func (v Version) Core() Version {
	c := v
	c.Pre = ""
	return c
}

// WithPre returns a copy of v carrying the given pre-release identifier.
func (v Version) WithPre(pre string) Version {
	c := v
	c.Pre = pre
	return c
}

// NextPre increments a trailing integer run in v.Pre, keeping the core numbers
// and prefix untouched:
//
//	rc.1    -> rc.2
//	rc      -> rc.1
//	beta.9  -> beta.10
//	alpha   -> alpha.1
//
// When Pre ends with a run of digits that run is incremented in place (any
// separator before it is preserved); otherwise ".1" is appended.
func (v Version) NextPre() Version {
	c := v
	pre := v.Pre
	i := len(pre)
	for i > 0 && pre[i-1] >= '0' && pre[i-1] <= '9' {
		i--
	}
	if i == len(pre) {
		c.Pre = pre + ".1"
		return c
	}
	n, err := strconv.Atoi(pre[i:])
	if err != nil {
		c.Pre = pre + ".1"
		return c
	}
	c.Pre = pre[:i] + strconv.Itoa(n+1)
	return c
}

// Compare orders a and b by CORE version only — Major, then Minor, then Patch —
// returning -1, 0, or +1. Pre-release identifiers are ignored: callers here
// only need core ordering.
func Compare(a, b Version) int {
	switch {
	case a.Major != b.Major:
		return sign(a.Major - b.Major)
	case a.Minor != b.Minor:
		return sign(a.Minor - b.Minor)
	case a.Patch != b.Patch:
		return sign(a.Patch - b.Patch)
	default:
		return 0
	}
}

func sign(n int) int {
	switch {
	case n < 0:
		return -1
	case n > 0:
		return 1
	default:
		return 0
	}
}

// Next returns the version after applying b, always as a stable core (no
// pre-release). None returns v unchanged.
func (v Version) Next(b Bump) Version {
	switch b {
	case Major:
		return Version{Major: v.Major + 1, Minor: 0, Patch: 0, Prefix: v.Prefix}
	case Minor:
		return Version{Major: v.Major, Minor: v.Minor + 1, Patch: 0, Prefix: v.Prefix}
	case Patch:
		return Version{Major: v.Major, Minor: v.Minor, Patch: v.Patch + 1, Prefix: v.Prefix}
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
