package cmd

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/soyagvs/relio/internal/config"
	"github.com/soyagvs/relio/internal/gitrepo"
	"github.com/soyagvs/relio/internal/i18n"
)

func undoGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	c := exec.Command("git", args...)
	c.Dir = dir
	if out, err := c.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v: %s", args, err, out)
	}
}

func undoWrite(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// undoReleaseState lays down a realistic "just released, not pushed" repo:
// an init commit tagged v1.0.0, a feature commit, then a changelog + release
// commit tagged v1.1.0 at HEAD.
func undoReleaseState(t *testing.T, dir string) {
	t.Helper()
	undoGit(t, dir, "commit", "--allow-empty", "-q", "-m", "chore: init")
	undoGit(t, dir, "tag", "v1.0.0")
	undoWrite(t, dir, "feature.go", "package x\n")
	undoGit(t, dir, "add", "feature.go")
	undoGit(t, dir, "commit", "-q", "-m", "feat: x")
	undoWrite(t, dir, "CHANGELOG.md", "# Changelog\n\n## v1.1.0\n")
	undoGit(t, dir, "add", "CHANGELOG.md")
	undoGit(t, dir, "commit", "-q", "-m", "chore(release): v1.1.0")
	undoGit(t, dir, "tag", "v1.1.0")
}

func TestRunUndoNoTags(t *testing.T) {
	dir, r := newStatusRepo(t)
	statusCommit(t, dir, "chore: init")

	var buf bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&buf)
	if err := runUndo(cmd, r, config.Default("proj"), true, false); err != nil {
		t.Fatalf("runUndo: %v", err)
	}
	if !strings.Contains(buf.String(), "nothing to undo") {
		t.Errorf("output missing 'nothing to undo':\n%s", buf.String())
	}
	if _, err := r.HeadSubject(); err != nil {
		t.Errorf("commit no longer present: %v", err)
	}
}

func TestRunUndoRemovesTagAndReleaseCommit(t *testing.T) {
	dir, r := newStatusRepo(t)
	undoReleaseState(t, dir)

	var buf bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&buf)
	if err := runUndo(cmd, r, config.Default("proj"), true, false); err != nil {
		t.Fatalf("runUndo: %v", err)
	}
	if has, _ := r.HasTag("v1.1.0"); has {
		t.Error("tag v1.1.0 still present")
	}
	subject, _ := r.HeadSubject()
	if subject != "feat: x" {
		t.Errorf("HeadSubject = %q, want %q", subject, "feat: x")
	}
	if _, err := os.Stat(filepath.Join(dir, "CHANGELOG.md")); !os.IsNotExist(err) {
		t.Errorf("CHANGELOG.md still present after undo: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "deleted tag v1.1.0") || !strings.Contains(out, "removed the release commit") {
		t.Errorf("output missing success lines:\n%s", out)
	}
}

func TestRunUndoTagNotAtHead(t *testing.T) {
	dir, r := newStatusRepo(t)
	undoReleaseState(t, dir)
	statusCommit(t, dir, "feat: later work")

	var buf bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&buf)
	err := runUndo(cmd, r, config.Default("proj"), true, false)
	if err == nil || !strings.Contains(err.Error(), "does not point at HEAD") {
		t.Fatalf("err = %v, want one mentioning 'does not point at HEAD'", err)
	}
}

func TestRunUndoRefusesWhenPushed(t *testing.T) {
	dir, r := newStatusRepo(t)
	undoReleaseState(t, dir)

	bare := t.TempDir()
	if out, err := exec.Command("git", "init", "--bare", "-q", bare).CombinedOutput(); err != nil {
		t.Fatalf("git init --bare: %v: %s", err, out)
	}
	branch, err := r.CurrentBranch()
	if err != nil {
		t.Fatal(err)
	}
	undoGit(t, dir, "remote", "add", "origin", bare)
	undoGit(t, dir, "push", "-u", "origin", branch)

	var buf bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&buf)
	err = runUndo(cmd, r, config.Default("proj"), true, false)
	if err == nil || !strings.Contains(err.Error(), "already on a remote") {
		t.Fatalf("err = %v, want one mentioning 'already on a remote'", err)
	}
}

func TestRunUndoDirtyTreeWithoutForce(t *testing.T) {
	dir, r := newStatusRepo(t)
	undoReleaseState(t, dir)
	undoWrite(t, dir, "feature.go", "package x\n// edited\n")

	var buf bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&buf)
	err := runUndo(cmd, r, config.Default("proj"), true, false)
	if err == nil || !strings.Contains(err.Error(), "uncommitted changes") {
		t.Fatalf("err = %v, want one mentioning 'uncommitted changes'", err)
	}

	var buf2 bytes.Buffer
	cmd2 := &cobra.Command{}
	cmd2.SetOut(&buf2)
	if err := runUndo(cmd2, r, config.Default("proj"), true, true); err != nil {
		t.Fatalf("runUndo --force: %v", err)
	}
	if has, _ := r.HasTag("v1.1.0"); has {
		t.Error("tag v1.1.0 still present after --force undo")
	}
	if subject, _ := r.HeadSubject(); subject != "feat: x" {
		t.Errorf("HeadSubject = %q, want %q", subject, "feat: x")
	}
}

func TestRunUndoCancelOnPrompt(t *testing.T) {
	dir, r := newStatusRepo(t)
	undoReleaseState(t, dir)

	var buf bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&buf)
	cmd.SetIn(strings.NewReader("n\n"))
	if err := runUndo(cmd, r, config.Default("proj"), false, false); err != nil {
		t.Fatalf("runUndo: %v", err)
	}
	if !strings.Contains(buf.String(), "Cancelled") {
		t.Errorf("output missing 'Cancelled':\n%s", buf.String())
	}
	if has, _ := r.HasTag("v1.1.0"); !has {
		t.Error("tag v1.1.0 was removed despite cancelling")
	}
	_ = dir
}

func TestRunUndoTagOnlyNoReleaseCommit(t *testing.T) {
	dir, r := newStatusRepo(t)
	undoGit(t, dir, "commit", "--allow-empty", "-q", "-m", "chore: init")
	undoWrite(t, dir, "feature.go", "package x\n")
	undoGit(t, dir, "add", "feature.go")
	undoGit(t, dir, "commit", "-q", "-m", "feat: x")
	undoGit(t, dir, "tag", "v1.1.0")

	var buf bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&buf)
	if err := runUndo(cmd, r, config.Default("proj"), true, false); err != nil {
		t.Fatalf("runUndo: %v", err)
	}
	if has, _ := r.HasTag("v1.1.0"); has {
		t.Error("tag v1.1.0 still present")
	}
	if subject, _ := r.HeadSubject(); subject != "feat: x" {
		t.Errorf("HeadSubject = %q, want %q", subject, "feat: x")
	}
	out := buf.String()
	if !strings.Contains(out, "deleted tag v1.1.0") {
		t.Errorf("output missing 'deleted tag v1.1.0':\n%s", out)
	}
	if strings.Contains(out, "removed the release commit") {
		t.Errorf("output should not mention removing a release commit:\n%s", out)
	}
}

// --- i18n: construction-time Short/flag usage + runtime output localization ---

func TestNewUndoCmdShortAndFlagsLocalizeAtConstructionTime(t *testing.T) {
	prev := i18n.Current()
	t.Cleanup(func() { i18n.SetLanguage(prev) })

	i18n.SetLanguage("en")
	cEN := newUndoCmd(&releaseFlags{})
	shortEN := cEN.Short
	yesUsageEN := cEN.Flags().Lookup("yes").Usage
	forceUsageEN := cEN.Flags().Lookup("force").Usage

	i18n.SetLanguage("es")
	cES := newUndoCmd(&releaseFlags{})
	shortES := cES.Short
	yesUsageES := cES.Flags().Lookup("yes").Usage
	forceUsageES := cES.Flags().Lookup("force").Usage

	if shortEN != "Reverse the most recent local release (before it is pushed)" {
		t.Errorf("newUndoCmd().Short (en) = %q", shortEN)
	}
	if shortES == shortEN || shortES == "" {
		t.Errorf("newUndoCmd().Short unchanged across languages: %q", shortES)
	}
	if yesUsageES == yesUsageEN || yesUsageES == "" {
		t.Errorf("--yes usage unchanged across languages: %q", yesUsageES)
	}
	if forceUsageES == forceUsageEN || forceUsageES == "" {
		t.Errorf("--force usage unchanged across languages: %q", forceUsageES)
	}
}

func TestRunUndoNoTagsLocalizesOutput(t *testing.T) {
	dir, r := newStatusRepo(t)
	statusCommit(t, dir, "chore: init")

	prev := i18n.Current()
	t.Cleanup(func() { i18n.SetLanguage(prev) })

	i18n.SetLanguage("en")
	var bufEN bytes.Buffer
	cmdEN := &cobra.Command{}
	cmdEN.SetOut(&bufEN)
	if err := runUndo(cmdEN, r, config.Default("proj"), true, false); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(bufEN.String(), "nothing to undo") {
		t.Errorf("english golden text missing:\n%s", bufEN.String())
	}

	i18n.SetLanguage("es")
	var bufES bytes.Buffer
	cmdES := &cobra.Command{}
	cmdES.SetOut(&bufES)
	if err := runUndo(cmdES, r, config.Default("proj"), true, false); err != nil {
		t.Fatal(err)
	}

	if bufEN.String() == bufES.String() {
		t.Error("runUndo no-tags output unchanged across languages")
	}
}

func TestRunUndoRemovesTagAndReleaseCommitLocalizesOutput(t *testing.T) {
	prev := i18n.Current()
	t.Cleanup(func() { i18n.SetLanguage(prev) })

	dirEN, rEN := newStatusRepo(t)
	undoReleaseState(t, dirEN)
	dirES, rES := newStatusRepo(t)
	undoReleaseState(t, dirES)
	_, _ = dirEN, dirES

	i18n.SetLanguage("en")
	var bufEN bytes.Buffer
	cmdEN := &cobra.Command{}
	cmdEN.SetOut(&bufEN)
	if err := runUndo(cmdEN, rEN, config.Default("proj"), true, false); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(bufEN.String(), "deleted tag v1.1.0") || !strings.Contains(bufEN.String(), "removed the release commit") {
		t.Errorf("english golden text missing:\n%s", bufEN.String())
	}

	i18n.SetLanguage("es")
	var bufES bytes.Buffer
	cmdES := &cobra.Command{}
	cmdES.SetOut(&bufES)
	if err := runUndo(cmdES, rES, config.Default("proj"), true, false); err != nil {
		t.Fatal(err)
	}

	if bufEN.String() == bufES.String() {
		t.Error("runUndo removed-tag output unchanged across languages")
	}
}

func TestRunUndoTagNotAtHeadLocalizesError(t *testing.T) {
	prev := i18n.Current()
	t.Cleanup(func() { i18n.SetLanguage(prev) })

	dirEN, rEN := newStatusRepo(t)
	undoReleaseState(t, dirEN)
	statusCommit(t, dirEN, "feat: later work")
	dirES, rES := newStatusRepo(t)
	undoReleaseState(t, dirES)
	statusCommit(t, dirES, "feat: later work")

	i18n.SetLanguage("en")
	cmdEN := &cobra.Command{}
	cmdEN.SetOut(io.Discard)
	errEN := runUndo(cmdEN, rEN, config.Default("proj"), true, false)
	i18n.SetLanguage("es")
	cmdES := &cobra.Command{}
	cmdES.SetOut(io.Discard)
	errES := runUndo(cmdES, rES, config.Default("proj"), true, false)

	if errEN == nil || errES == nil {
		t.Fatal("expected does-not-point-at-HEAD errors in both languages")
	}
	if !strings.Contains(errEN.Error(), "does not point at HEAD") {
		t.Errorf("english golden error text missing: %q", errEN.Error())
	}
	if errEN.Error() == errES.Error() {
		t.Errorf("does-not-point-at-HEAD error unchanged across languages: %q", errEN.Error())
	}
}

func TestRunUndoRefusesWhenPushedLocalizesError(t *testing.T) {
	prev := i18n.Current()
	t.Cleanup(func() { i18n.SetLanguage(prev) })

	setup := func(t *testing.T) (dir string, r *gitrepo.Repo) {
		dir, r = newStatusRepo(t)
		undoReleaseState(t, dir)
		bare := t.TempDir()
		if out, err := exec.Command("git", "init", "--bare", "-q", bare).CombinedOutput(); err != nil {
			t.Fatalf("git init --bare: %v: %s", err, out)
		}
		branch, err := r.CurrentBranch()
		if err != nil {
			t.Fatal(err)
		}
		undoGit(t, dir, "remote", "add", "origin", bare)
		undoGit(t, dir, "push", "-u", "origin", branch)
		return dir, r
	}

	_, rEN := setup(t)
	_, rES := setup(t)

	i18n.SetLanguage("en")
	cmdEN := &cobra.Command{}
	cmdEN.SetOut(io.Discard)
	errEN := runUndo(cmdEN, rEN, config.Default("proj"), true, false)
	i18n.SetLanguage("es")
	cmdES := &cobra.Command{}
	cmdES.SetOut(io.Discard)
	errES := runUndo(cmdES, rES, config.Default("proj"), true, false)

	if errEN == nil || errES == nil {
		t.Fatal("expected already-on-a-remote errors in both languages")
	}
	if !strings.Contains(errEN.Error(), "already on a remote") {
		t.Errorf("english golden error text missing: %q", errEN.Error())
	}
	if errEN.Error() == errES.Error() {
		t.Errorf("already-on-a-remote error unchanged across languages: %q", errEN.Error())
	}
}

func TestRunUndoDirtyTreeLocalizesError(t *testing.T) {
	prev := i18n.Current()
	t.Cleanup(func() { i18n.SetLanguage(prev) })

	dirEN, rEN := newStatusRepo(t)
	undoReleaseState(t, dirEN)
	undoWrite(t, dirEN, "feature.go", "package x\n// edited\n")
	dirES, rES := newStatusRepo(t)
	undoReleaseState(t, dirES)
	undoWrite(t, dirES, "feature.go", "package x\n// edited\n")

	i18n.SetLanguage("en")
	cmdEN := &cobra.Command{}
	cmdEN.SetOut(io.Discard)
	errEN := runUndo(cmdEN, rEN, config.Default("proj"), true, false)
	i18n.SetLanguage("es")
	cmdES := &cobra.Command{}
	cmdES.SetOut(io.Discard)
	errES := runUndo(cmdES, rES, config.Default("proj"), true, false)

	if errEN == nil || errES == nil {
		t.Fatal("expected dirty-tree errors in both languages")
	}
	if !strings.Contains(errEN.Error(), "uncommitted changes") {
		t.Errorf("english golden error text missing: %q", errEN.Error())
	}
	if errEN.Error() == errES.Error() {
		t.Errorf("dirty-tree error unchanged across languages: %q", errEN.Error())
	}
}

func TestRunUndoCancelOnPromptLocalizesOutput(t *testing.T) {
	prev := i18n.Current()
	t.Cleanup(func() { i18n.SetLanguage(prev) })

	dirEN, rEN := newStatusRepo(t)
	undoReleaseState(t, dirEN)
	dirES, rES := newStatusRepo(t)
	undoReleaseState(t, dirES)
	_, _ = dirEN, dirES

	i18n.SetLanguage("en")
	var bufEN bytes.Buffer
	cmdEN := &cobra.Command{}
	cmdEN.SetOut(&bufEN)
	cmdEN.SetIn(strings.NewReader("n\n"))
	if err := runUndo(cmdEN, rEN, config.Default("proj"), false, false); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(bufEN.String(), "Cancelled") {
		t.Errorf("english golden text missing:\n%s", bufEN.String())
	}

	i18n.SetLanguage("es")
	var bufES bytes.Buffer
	cmdES := &cobra.Command{}
	cmdES.SetOut(&bufES)
	cmdES.SetIn(strings.NewReader("n\n"))
	if err := runUndo(cmdES, rES, config.Default("proj"), false, false); err != nil {
		t.Fatal(err)
	}

	if bufEN.String() == bufES.String() {
		t.Error("runUndo cancelled output unchanged across languages")
	}
}
