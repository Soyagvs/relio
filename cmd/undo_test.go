package cmd

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/soyagvs/relio/internal/config"
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
