package cmd

import (
	"strings"
	"testing"
)

func TestQRBlockRendersAndIndents(t *testing.T) {
	out := qrBlock("https://0x0.st/abcd.png")
	if out == "" {
		t.Fatal("empty QR")
	}
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) < 15 {
		t.Errorf("QR only %d lines, expected a full matrix", len(lines))
	}
	for i, ln := range lines {
		if !strings.HasPrefix(ln, "  ") {
			t.Errorf("line %d not indented: %q", i, ln)
		}
		if !strings.Contains(ln, "\x1b[4") {
			t.Errorf("line %d has no ANSI cell: %q", i, ln)
		}
	}
}

func TestImageDest(t *testing.T) {
	cases := []struct {
		name            string
		im              imageFlags
		interactive     bool
		save, link, ask bool
	}{
		{"no flags, no tty -> save only", imageFlags{}, false, true, false, false},
		{"no flags, tty -> ask", imageFlags{}, true, false, false, true},
		{"--upload -> save + link", imageFlags{upload: true}, true, true, true, false},
		{"--upload, no tty -> save + link", imageFlags{upload: true}, false, true, true, false},
		{"--link-only -> link only", imageFlags{linkOnly: true}, true, false, true, false},
		{"--link-only beats --upload", imageFlags{upload: true, linkOnly: true}, true, false, true, false},
	}
	for _, tc := range cases {
		s, l, a := imageDest(tc.im, tc.interactive)
		if s != tc.save || l != tc.link || a != tc.ask {
			t.Errorf("%s: got save=%v link=%v ask=%v, want %v/%v/%v",
				tc.name, s, l, a, tc.save, tc.link, tc.ask)
		}
	}
}

func TestDestFromAnswer(t *testing.T) {
	for _, tc := range []struct {
		ans        string
		save, link bool
	}{
		{"both", true, true},
		{"save", true, false},
		{"link", false, true},
	} {
		if s, l := destFromAnswer(tc.ans); s != tc.save || l != tc.link {
			t.Errorf("%q: got save=%v link=%v, want %v/%v", tc.ans, s, l, tc.save, tc.link)
		}
	}
}

func TestImageMeta(t *testing.T) {
	if got := imageMeta("2026-09-06 13:47", "2026-09-06", 4); got != "06.09.26 · 13:47 · 4 commits" {
		t.Errorf("imageMeta full = %q", got)
	}
	if got := imageMeta("", "2026-09-06", 2); got != "2026-09-06 · 2 commits" {
		t.Errorf("imageMeta date-only = %q", got)
	}
}
