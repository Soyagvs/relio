package ui

import (
	"strings"
	"testing"

	"github.com/soyagvs/relio/internal/changelog"
)

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
