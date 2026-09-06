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

func TestImageMeta(t *testing.T) {
	if got := imageMeta("2026-09-06 13:47", "2026-09-06", 4); got != "06.09.26 · 13:47 · 4 commits" {
		t.Errorf("imageMeta full = %q", got)
	}
	if got := imageMeta("", "2026-09-06", 2); got != "2026-09-06 · 2 commits" {
		t.Errorf("imageMeta date-only = %q", got)
	}
}
