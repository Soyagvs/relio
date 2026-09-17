package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/soyagvs/relio/internal/config"
	"github.com/soyagvs/relio/internal/i18n"
)

// releasesWrite is a small os.WriteFile wrapper matching the style of
// undoWrite in cmd/undo_test.go.
func releasesWrite(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// releasesEditorScript writes a fake editor shell script that replaces the
// passed file's contents with body, matching the technique used in
// internal/editor/editor_test.go.
func releasesEditorScript(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "fake-editor.sh")
	script := "#!/bin/sh\nprintf '%s' " + shQuote(body) + " > \"$1\"\n"
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

// shQuote wraps s in single quotes for embedding in a POSIX shell script,
// escaping any single quotes it contains.
func shQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

const releasesChangelogFixture = "# Changelog\n\n" +
	"## [1.1.0] - 2024-02-01\n\n### Added\n\n- New thing\n\n" +
	"## [1.0.0] - 2024-01-01\n\n### Added\n\n- Typo hree\n"

func TestRunReleasesEditReplacesSectionOnly(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake editor is a POSIX shell script")
	}
	dir, r := newStatusRepo(t)
	releasesWrite(t, dir, "CHANGELOG.md", releasesChangelogFixture)

	script := releasesEditorScript(t, "## [1.0.0] - 2024-01-01\n\n### Added\n\n- Typo here\n")
	t.Setenv("RELIO_EDITOR", script)

	var buf bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&buf)
	if err := runReleasesEdit(cmd, r, config.Default("proj"), "1.0.0"); err != nil {
		t.Fatalf("runReleasesEdit: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(dir, "CHANGELOG.md"))
	if err != nil {
		t.Fatal(err)
	}
	gotStr := string(got)
	if !strings.Contains(gotStr, "- Typo here") {
		t.Errorf("edited text not written:\n%s", gotStr)
	}
	if strings.Contains(gotStr, "- Typo hree") {
		t.Errorf("old text still present:\n%s", gotStr)
	}
	if !strings.Contains(gotStr, "## [1.1.0] - 2024-02-01") || !strings.Contains(gotStr, "- New thing") {
		t.Errorf("untouched section was modified:\n%s", gotStr)
	}
	if !strings.Contains(buf.String(), "CHANGELOG.md") {
		t.Errorf("output missing done message:\n%s", buf.String())
	}
}

func TestRunReleasesEditNoSuchVersion(t *testing.T) {
	dir, r := newStatusRepo(t)
	releasesWrite(t, dir, "CHANGELOG.md", releasesChangelogFixture)

	err := runReleasesEdit(&cobra.Command{}, r, config.Default("proj"), "9.9.9")
	if err == nil {
		t.Fatal("runReleasesEdit: error = nil, want non-nil for missing version")
	}

	got, rerr := os.ReadFile(filepath.Join(dir, "CHANGELOG.md"))
	if rerr != nil {
		t.Fatal(rerr)
	}
	if string(got) != releasesChangelogFixture {
		t.Errorf("changelog modified despite error:\n%s", got)
	}
}

func TestRunReleasesEditNoChangelogFile(t *testing.T) {
	_, r := newStatusRepo(t)

	err := runReleasesEdit(&cobra.Command{}, r, config.Default("proj"), "1.0.0")
	if err == nil {
		t.Fatal("runReleasesEdit: error = nil, want non-nil when no changelog exists")
	}
}

func TestRunReleasesEditUnchangedDoesNotWrite(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake editor is a POSIX shell script")
	}
	dir, r := newStatusRepo(t)
	releasesWrite(t, dir, "CHANGELOG.md", releasesChangelogFixture)

	// The fake editor writes back exactly the extracted section, unchanged.
	script := releasesEditorScript(t, "## [1.0.0] - 2024-01-01\n\n### Added\n\n- Typo hree")
	t.Setenv("RELIO_EDITOR", script)

	var buf bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&buf)
	if err := runReleasesEdit(cmd, r, config.Default("proj"), "1.0.0"); err != nil {
		t.Fatalf("runReleasesEdit: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(dir, "CHANGELOG.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != releasesChangelogFixture {
		t.Errorf("changelog file modified when editor returned unchanged content:\n%s", got)
	}
	if !strings.Contains(buf.String(), "Unchanged.") {
		t.Errorf("output missing 'Unchanged.':\n%s", buf.String())
	}
}

func TestRunReleasesEditBlankRefused(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake editor is a POSIX shell script")
	}
	dir, r := newStatusRepo(t)
	releasesWrite(t, dir, "CHANGELOG.md", releasesChangelogFixture)

	script := releasesEditorScript(t, "")
	t.Setenv("RELIO_EDITOR", script)

	err := runReleasesEdit(&cobra.Command{}, r, config.Default("proj"), "1.0.0")
	if err == nil {
		t.Fatal("runReleasesEdit: error = nil, want non-nil when editor returns blank content")
	}

	got, rerr := os.ReadFile(filepath.Join(dir, "CHANGELOG.md"))
	if rerr != nil {
		t.Fatal(rerr)
	}
	if string(got) != releasesChangelogFixture {
		t.Errorf("changelog modified despite blank-content error:\n%s", got)
	}
}

func TestNewReleasesCmdShortLocalizeAtConstructionTime(t *testing.T) {
	prev := i18n.Current()
	t.Cleanup(func() { i18n.SetLanguage(prev) })

	i18n.SetLanguage("en")
	groupEN := newReleasesCmd(&releaseFlags{})
	editEN, _, _ := groupEN.Find([]string{"edit"})

	if groupEN.Short != "Work with existing releases" {
		t.Errorf("newReleasesCmd().Short (en) = %q", groupEN.Short)
	}
	if editEN.Short != "Fix a typo in an already-published changelog section" {
		t.Errorf("releases edit Short (en) = %q", editEN.Short)
	}

	i18n.SetLanguage("es")
	groupES := newReleasesCmd(&releaseFlags{})
	editES, _, _ := groupES.Find([]string{"edit"})

	if groupES.Short == groupEN.Short || groupES.Short == "" {
		t.Errorf("newReleasesCmd().Short unchanged across languages: %q", groupES.Short)
	}
	if editES.Short == editEN.Short || editES.Short == "" {
		t.Errorf("releases edit Short unchanged across languages: %q", editES.Short)
	}
}
