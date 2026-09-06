// Package conventional parses Conventional Commits messages into structured data.
//
// Reference: https://www.conventionalcommits.org/en/v1.0.0/
package conventional

import (
	"strings"
)

// Commit is a single parsed commit.
type Commit struct {
	Hash        string
	Type        string // normalized lowercase type, e.g. "feat", "fix". Empty when not conventional.
	Scope       string
	Description string
	Body        string
	Breaking    bool
	// Raw holds the original subject line, used as a fallback when Type is empty.
	Raw string
}

// IsConventional reports whether the commit followed the Conventional Commits format.
func (c Commit) IsConventional() bool { return c.Type != "" }

// headerPrefix splits "type(scope)!" from the rest. It returns ok=false when the
// subject has no "type: description" shape.
func splitHeader(subject string) (typ, scope string, breaking bool, desc string, ok bool) {
	idx := strings.Index(subject, ":")
	if idx < 0 {
		return "", "", false, "", false
	}
	left := strings.TrimSpace(subject[:idx])
	desc = strings.TrimSpace(subject[idx+1:])
	if left == "" || desc == "" {
		return "", "", false, "", false
	}

	if strings.HasSuffix(left, "!") {
		breaking = true
		left = strings.TrimSpace(strings.TrimSuffix(left, "!"))
	}

	if open := strings.Index(left, "("); open >= 0 {
		close := strings.Index(left, ")")
		if close < open {
			return "", "", false, "", false
		}
		scope = strings.TrimSpace(left[open+1 : close])
		left = strings.TrimSpace(left[:open])
	}

	// A valid type is a single token of letters.
	if left == "" || strings.ContainsAny(left, " \t") {
		return "", "", false, "", false
	}
	for _, r := range left {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')) {
			return "", "", false, "", false
		}
	}
	return strings.ToLower(left), scope, breaking, desc, true
}

// Parse builds a Commit from its hash, subject line, and body.
func Parse(hash, subject, body string) Commit {
	subject = strings.TrimSpace(subject)
	body = strings.TrimRight(body, "\n\r \t")

	c := Commit{Hash: hash, Body: body, Raw: subject}

	typ, scope, breaking, desc, ok := splitHeader(subject)
	if ok {
		c.Type = typ
		c.Scope = scope
		c.Breaking = breaking
		c.Description = desc
	} else {
		c.Description = subject
	}

	if bodyDeclaresBreaking(body) {
		c.Breaking = true
	}
	return c
}

// bodyDeclaresBreaking looks for the "BREAKING CHANGE:" / "BREAKING-CHANGE:"
// footer token anywhere in the body.
func bodyDeclaresBreaking(body string) bool {
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "BREAKING CHANGE:") || strings.HasPrefix(line, "BREAKING-CHANGE:") {
			return true
		}
	}
	return false
}

// ParseMany parses a slice of raw commits.
func ParseMany(raw []Raw) []Commit {
	out := make([]Commit, 0, len(raw))
	for _, r := range raw {
		out = append(out, Parse(r.Hash, r.Subject, r.Body))
	}
	return out
}

// Raw is an unparsed commit as read from git.
type Raw struct {
	Hash    string
	Subject string
	Body    string
}
