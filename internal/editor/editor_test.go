package editor

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// writeScript writes an executable shell script to a temp dir and returns its path.
func writeScript(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "fake-editor.sh")
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestEditReturnsEditedContent(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake editor is a POSIX shell script")
	}
	script := writeScript(t, "#!/bin/sh\nprintf '%s\\n' \"HAND EDITED\" > \"$1\"\n")
	t.Setenv("RELIO_EDITOR", script)

	got, err := Edit("original")
	if err != nil {
		t.Fatalf("Edit: %v", err)
	}
	if !strings.Contains(got, "HAND EDITED") {
		t.Errorf("Edit() = %q, want it to contain %q", got, "HAND EDITED")
	}
}

func TestEditResolutionPrecedence(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake editor is a POSIX shell script")
	}
	scriptA := writeScript(t, "#!/bin/sh\nprintf '%s\\n' \"FROM RELIO\" > \"$1\"\n")
	scriptB := writeScript(t, "#!/bin/sh\nprintf '%s\\n' \"FROM EDITOR\" > \"$1\"\n")
	t.Setenv("RELIO_EDITOR", scriptA)
	t.Setenv("EDITOR", scriptB)

	got, err := Edit("original")
	if err != nil {
		t.Fatalf("Edit: %v", err)
	}
	if !strings.Contains(got, "FROM RELIO") {
		t.Errorf("Edit() = %q, want RELIO_EDITOR to take precedence over EDITOR", got)
	}
}

func TestEditEditorFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("relies on the POSIX `false` binary")
	}
	t.Setenv("RELIO_EDITOR", "false")
	if _, err := Edit("original"); err == nil {
		t.Fatal("Edit() error = nil, want non-nil when the editor exits non-zero")
	}
}

func TestEditSeedsFileWithText(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake editor is a POSIX shell script")
	}
	capture := filepath.Join(t.TempDir(), "capture.md")
	script := writeScript(t, "#!/bin/sh\ncp \"$1\" \"$RELIO_TEST_CAPTURE\"\n")
	t.Setenv("RELIO_EDITOR", script)
	t.Setenv("RELIO_TEST_CAPTURE", capture)

	if _, err := Edit("seed content here"); err != nil {
		t.Fatalf("Edit: %v", err)
	}
	data, err := os.ReadFile(capture)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "seed content here") {
		t.Errorf("seeded file = %q, want it to contain %q", data, "seed content here")
	}
}
