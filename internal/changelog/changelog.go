// Package changelog renders release notes and maintains a CHANGELOG.md file in
// the "Keep a Changelog" style (https://keepachangelog.com/en/1.1.0/).
package changelog

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/soyagvs/go-release/internal/conventional"
)

// Header is written when a CHANGELOG.md does not exist yet.
const Header = `# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0/).
`

// Group is a Keep a Changelog section name.
type Group string

const (
	Added      Group = "Added"
	Changed    Group = "Changed"
	Fixed      Group = "Fixed"
	Removed    Group = "Removed"
	Deprecated Group = "Deprecated"
	Security   Group = "Security"
)

// groupOrder is the canonical rendering order.
var groupOrder = []Group{Added, Changed, Deprecated, Removed, Fixed, Security}

// groupFor maps a conventional commit type to a changelog group. The bool is
// false when the type should be omitted from the changelog.
func groupFor(typ string) (Group, bool) {
	switch typ {
	case "feat":
		return Added, true
	case "fix":
		return Fixed, true
	case "perf", "refactor", "revert", "style":
		return Changed, true
	default:
		// docs, chore, test, build, ci, and unknown types are omitted.
		return "", false
	}
}

// Notes is the grouped set of human-readable lines for one release.
type Notes struct {
	Groups map[Group][]string
}

// Empty reports whether there is nothing worth publishing.
func (n Notes) Empty() bool {
	for _, v := range n.Groups {
		if len(v) > 0 {
			return false
		}
	}
	return true
}

// Build turns commits into grouped notes. Breaking changes are prefixed and
// always land in the Changed group in addition to their natural group.
func Build(commits []conventional.Commit) Notes {
	n := Notes{Groups: map[Group][]string{}}
	for _, c := range commits {
		line := lineFor(c)
		if c.Breaking {
			n.Groups[Changed] = append(n.Groups[Changed], "**Breaking:** "+line)
			continue
		}
		g, ok := groupFor(c.Type)
		if !ok {
			continue
		}
		n.Groups[g] = append(n.Groups[g], line)
	}
	return n
}

func lineFor(c conventional.Commit) string {
	desc := c.Description
	if desc == "" {
		desc = c.Raw
	}
	desc = strings.TrimSpace(desc)
	if len(desc) > 0 {
		desc = strings.ToUpper(desc[:1]) + desc[1:]
	}
	if c.Scope != "" {
		return fmt.Sprintf("%s: %s", c.Scope, desc)
	}
	return desc
}

// RenderSection renders a single "## [version] - date" block, without a trailing
// blank line.
func RenderSection(version string, date time.Time, n Notes) string {
	var b strings.Builder
	fmt.Fprintf(&b, "## [%s] - %s\n", strings.TrimPrefix(version, "v"), date.Format("2006-01-02"))

	wrote := false
	for _, g := range groupOrder {
		items := n.Groups[g]
		if len(items) == 0 {
			continue
		}
		sort.Strings(items)
		fmt.Fprintf(&b, "\n### %s\n\n", g)
		for _, it := range items {
			fmt.Fprintf(&b, "- %s\n", it)
		}
		wrote = true
	}
	if !wrote {
		b.WriteString("\n_No user-facing changes._\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

// Update inserts section at the top of the release history in existing content.
// When existing is empty it seeds the file with Header. The returned string ends
// with a single trailing newline.
func Update(existing, section string) string {
	existing = strings.TrimRight(existing, "\n")
	if strings.TrimSpace(existing) == "" {
		return Header + "\n" + section + "\n"
	}

	lines := strings.Split(existing, "\n")
	insertAt := -1
	for i, ln := range lines {
		if strings.HasPrefix(strings.TrimSpace(ln), "## [") {
			insertAt = i
			break
		}
	}

	if insertAt < 0 {
		// No prior releases: append after the header block.
		return existing + "\n\n" + section + "\n"
	}

	before := strings.Join(lines[:insertAt], "\n")
	after := strings.Join(lines[insertAt:], "\n")
	before = strings.TrimRight(before, "\n")
	return before + "\n\n" + section + "\n\n" + after + "\n"
}

// sectionBounds returns the [start, end) line indices of the "## [version]" block
// in lines, or ok=false when it is not present. version may be given with or
// without a leading "v".
func sectionBounds(lines []string, version string) (start, end int, ok bool) {
	want := "## [" + strings.TrimPrefix(version, "v") + "]"
	start = -1
	for i, ln := range lines {
		if strings.HasPrefix(strings.TrimSpace(ln), want) {
			start = i
			break
		}
	}
	if start < 0 {
		return 0, 0, false
	}
	end = len(lines)
	for i := start + 1; i < len(lines); i++ {
		if strings.HasPrefix(strings.TrimSpace(lines[i]), "## [") {
			end = i
			break
		}
	}
	return start, end, true
}

// ExtractSection returns the "## [version]" block from content, trimmed, or ""
// when that version has no section.
func ExtractSection(content, version string) string {
	lines := strings.Split(content, "\n")
	start, end, ok := sectionBounds(lines, version)
	if !ok {
		return ""
	}
	return strings.TrimRight(strings.Join(lines[start:end], "\n"), "\n ")
}

// RemoveSection deletes the "## [version]" block from content. When the version
// is absent, content is returned unchanged (aside from newline normalisation).
func RemoveSection(content, version string) string {
	lines := strings.Split(strings.TrimRight(content, "\n"), "\n")
	start, end, ok := sectionBounds(lines, version)
	if !ok {
		return strings.TrimRight(content, "\n") + "\n"
	}
	kept := append([]string{}, lines[:start]...)
	kept = append(kept, lines[end:]...)

	out := strings.Join(kept, "\n")
	// Collapse the 3+ newline gap left where the section was.
	for strings.Contains(out, "\n\n\n") {
		out = strings.ReplaceAll(out, "\n\n\n", "\n\n")
	}
	return strings.TrimRight(out, "\n") + "\n"
}
