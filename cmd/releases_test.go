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

// releasesTagState lays down two tagged commits, newest last: v1.0.0 then
// v1.1.0 at HEAD.
func releasesTagState(t *testing.T, dir string) {
	t.Helper()
	undoGit(t, dir, "commit", "--allow-empty", "-q", "-m", "chore: init")
	undoGit(t, dir, "tag", "v1.0.0")
	undoGit(t, dir, "commit", "--allow-empty", "-q", "-m", "chore: two")
	undoGit(t, dir, "tag", "v1.1.0")
}

const releasesDeleteChangelogFixture = "# Changelog\n\n" +
	"## [1.1.0] - 2024-02-01\n\n### Added\n\n- New thing\n\n" +
	"## [1.0.0] - 2024-01-01\n\n### Added\n\n- Old thing\n"

// --- releases list ---

func TestRunReleasesListPrintsAllTags(t *testing.T) {
	dir, r := newStatusRepo(t)
	releasesTagState(t, dir)

	var buf bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&buf)
	if err := runReleasesList(cmd, r); err != nil {
		t.Fatalf("runReleasesList: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "v1.1.0") || !strings.Contains(out, "v1.0.0") {
		t.Errorf("output missing a tag:\n%s", out)
	}
}

func TestRunReleasesListEmptyRepo(t *testing.T) {
	_, r := newStatusRepo(t)

	var buf bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&buf)
	if err := runReleasesList(cmd, r); err != nil {
		t.Fatalf("runReleasesList: %v", err)
	}
	if !strings.Contains(buf.String(), "No releases yet") {
		t.Errorf("output missing empty-state message:\n%s", buf.String())
	}
}

// --- releases delete ---

func TestRunReleasesDeleteRemovesTagAndChangelogSection(t *testing.T) {
	dir, r := newStatusRepo(t)
	releasesTagState(t, dir)
	releasesWrite(t, dir, "CHANGELOG.md", releasesDeleteChangelogFixture)

	var buf bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&buf)
	if err := runReleasesDelete(cmd, r, config.Default("proj"), "v1.1.0", true); err != nil {
		t.Fatalf("runReleasesDelete: %v", err)
	}
	if has, _ := r.HasTag("v1.1.0"); has {
		t.Error("tag v1.1.0 still present")
	}
	if has, _ := r.HasTag("v1.0.0"); !has {
		t.Error("unrelated tag v1.0.0 should remain")
	}

	got, err := os.ReadFile(filepath.Join(dir, "CHANGELOG.md"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(got), "## [1.1.0]") || strings.Contains(string(got), "New thing") {
		t.Errorf("changelog section not removed:\n%s", got)
	}
	if !strings.Contains(string(got), "## [1.0.0]") {
		t.Errorf("wrong section removed:\n%s", got)
	}
	if !strings.Contains(buf.String(), "tag + changelog section") {
		t.Errorf("output missing done message:\n%s", buf.String())
	}
}

// TestRunReleasesDeleteRecoversStaleChangelogSectionWhenTagAlreadyGone
// guards against a real gap: if a prior `releases delete` removed the tag
// but failed before cleaning up the changelog (e.g. a transient write
// failure), resolveTagName can no longer find a tag for that version, so a
// naive re-run would fail with "no such version" and leave the stale
// section stuck forever with no tool-mediated way to finish the cleanup.
func TestRunReleasesDeleteRecoversStaleChangelogSectionWhenTagAlreadyGone(t *testing.T) {
	dir, r := newStatusRepo(t)
	releasesTagState(t, dir)
	releasesWrite(t, dir, "CHANGELOG.md", releasesDeleteChangelogFixture)
	// Simulate the interrupted-delete state directly: the tag is already
	// gone, but the changelog section for it was never removed.
	undoGit(t, dir, "tag", "-d", "v1.1.0")

	var buf bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&buf)
	if err := runReleasesDelete(cmd, r, config.Default("proj"), "v1.1.0", true); err != nil {
		t.Fatalf("runReleasesDelete: %v, want it to recover by finishing the stale changelog cleanup", err)
	}

	got, err := os.ReadFile(filepath.Join(dir, "CHANGELOG.md"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(got), "## [1.1.0]") || strings.Contains(string(got), "New thing") {
		t.Errorf("stale changelog section not removed:\n%s", got)
	}
	if !strings.Contains(string(got), "## [1.0.0]") {
		t.Errorf("wrong section removed:\n%s", got)
	}
}

// TestRunReleasesDeleteNoSuchVersionWhenNothingToRecover proves the
// recovery path only kicks in when there's actually a stale section to
// clean up — a version with neither a tag nor a changelog section still
// correctly errors as "no such version."
func TestRunReleasesDeleteNoSuchVersionWhenNothingToRecover(t *testing.T) {
	dir, r := newStatusRepo(t)
	releasesTagState(t, dir)

	err := runReleasesDelete(&cobra.Command{}, r, config.Default("proj"), "9.9.9", true)
	if err == nil {
		t.Fatal("runReleasesDelete: error = nil, want non-nil when there is truly nothing for this version")
	}
	_ = dir
}

func TestRunReleasesDeleteAcceptsVersionWithoutVPrefix(t *testing.T) {
	dir, r := newStatusRepo(t)
	releasesTagState(t, dir)

	var buf bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&buf)
	if err := runReleasesDelete(cmd, r, config.Default("proj"), "1.1.0", true); err != nil {
		t.Fatalf("runReleasesDelete: %v", err)
	}
	if has, _ := r.HasTag("v1.1.0"); has {
		t.Error("tag v1.1.0 still present")
	}
	_ = dir
}

func TestRunReleasesDeleteNoSuchVersion(t *testing.T) {
	dir, r := newStatusRepo(t)
	releasesTagState(t, dir)

	err := runReleasesDelete(&cobra.Command{}, r, config.Default("proj"), "9.9.9", true)
	if err == nil {
		t.Fatal("runReleasesDelete: error = nil, want non-nil for a missing tag")
	}
	if has, _ := r.HasTag("v1.1.0"); !has {
		t.Error("unrelated tag v1.1.0 should remain untouched")
	}
	_ = dir
}

func TestRunReleasesDeleteCancelOnPrompt(t *testing.T) {
	dir, r := newStatusRepo(t)
	releasesTagState(t, dir)

	var buf bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&buf)
	cmd.SetIn(strings.NewReader("n\n"))
	if err := runReleasesDelete(cmd, r, config.Default("proj"), "v1.1.0", false); err != nil {
		t.Fatalf("runReleasesDelete: %v", err)
	}
	if !strings.Contains(buf.String(), "Delete cancelled") {
		t.Errorf("output missing cancellation message:\n%s", buf.String())
	}
	if has, _ := r.HasTag("v1.1.0"); !has {
		t.Error("tag v1.1.0 was removed despite cancelling")
	}
	_ = dir
}

func TestRunReleasesDeletePromptsAndConfirms(t *testing.T) {
	dir, r := newStatusRepo(t)
	releasesTagState(t, dir)

	var buf bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&buf)
	cmd.SetIn(strings.NewReader("y\n"))
	if err := runReleasesDelete(cmd, r, config.Default("proj"), "v1.1.0", false); err != nil {
		t.Fatalf("runReleasesDelete: %v", err)
	}
	if has, _ := r.HasTag("v1.1.0"); has {
		t.Error("tag v1.1.0 still present after confirming")
	}
	if !strings.Contains(buf.String(), "Delete v1.1.0?") {
		t.Errorf("output missing confirmation prompt:\n%s", buf.String())
	}
	_ = dir
}

// --- i18n: construction-time Short/flag usage + runtime output localization ---

func TestNewReleasesListAndDeleteCmdShortAndFlagsLocalizeAtConstructionTime(t *testing.T) {
	prev := i18n.Current()
	t.Cleanup(func() { i18n.SetLanguage(prev) })

	i18n.SetLanguage("en")
	groupEN := newReleasesCmd(&releaseFlags{})
	listEN, _, _ := groupEN.Find([]string{"list"})
	delEN, _, _ := groupEN.Find([]string{"delete"})
	yesUsageEN := delEN.Flags().Lookup("yes").Usage

	i18n.SetLanguage("es")
	groupES := newReleasesCmd(&releaseFlags{})
	listES, _, _ := groupES.Find([]string{"list"})
	delES, _, _ := groupES.Find([]string{"delete"})
	yesUsageES := delES.Flags().Lookup("yes").Usage

	if listEN.Short != "List every local release tag" {
		t.Errorf("releases list Short (en) = %q", listEN.Short)
	}
	if listES.Short == listEN.Short || listES.Short == "" {
		t.Errorf("releases list Short unchanged across languages: %q", listES.Short)
	}
	if delEN.Short != "Delete a local release tag and its changelog section" {
		t.Errorf("releases delete Short (en) = %q", delEN.Short)
	}
	if delES.Short == delEN.Short || delES.Short == "" {
		t.Errorf("releases delete Short unchanged across languages: %q", delES.Short)
	}
	if yesUsageES == yesUsageEN || yesUsageES == "" {
		t.Errorf("--yes usage unchanged across languages: %q", yesUsageES)
	}
}

func TestRunReleasesDeleteNoSuchVersionLocalizesError(t *testing.T) {
	prev := i18n.Current()
	t.Cleanup(func() { i18n.SetLanguage(prev) })

	dirEN, rEN := newStatusRepo(t)
	releasesTagState(t, dirEN)
	dirES, rES := newStatusRepo(t)
	releasesTagState(t, dirES)
	_, _ = dirEN, dirES

	i18n.SetLanguage("en")
	errEN := runReleasesDelete(&cobra.Command{}, rEN, config.Default("proj"), "9.9.9", true)
	i18n.SetLanguage("es")
	errES := runReleasesDelete(&cobra.Command{}, rES, config.Default("proj"), "9.9.9", true)

	if errEN == nil || errES == nil {
		t.Fatal("expected no-such-tag errors in both languages")
	}
	if errEN.Error() == errES.Error() {
		t.Errorf("no-such-tag error unchanged across languages: %q", errEN.Error())
	}
}

func TestRunReleasesListEmptyLocalizesOutput(t *testing.T) {
	prev := i18n.Current()
	t.Cleanup(func() { i18n.SetLanguage(prev) })

	_, rEN := newStatusRepo(t)
	_, rES := newStatusRepo(t)

	i18n.SetLanguage("en")
	var bufEN bytes.Buffer
	cmdEN := &cobra.Command{}
	cmdEN.SetOut(&bufEN)
	if err := runReleasesList(cmdEN, rEN); err != nil {
		t.Fatal(err)
	}

	i18n.SetLanguage("es")
	var bufES bytes.Buffer
	cmdES := &cobra.Command{}
	cmdES.SetOut(&bufES)
	if err := runReleasesList(cmdES, rES); err != nil {
		t.Fatal(err)
	}

	if bufEN.String() == bufES.String() {
		t.Error("runReleasesList empty output unchanged across languages")
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
