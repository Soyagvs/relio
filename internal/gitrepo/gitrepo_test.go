package gitrepo

import (
	"os/exec"
	"testing"
)

func gitInit(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, args := range [][]string{
		{"init", "-q"},
		{"config", "user.email", "test@example.com"},
		{"config", "user.name", "Test"},
		{"config", "commit.gpgsign", "false"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	return dir
}

func commit(t *testing.T, dir, msg string) {
	t.Helper()
	cmd := exec.Command("git", "commit", "--allow-empty", "-q", "-m", msg)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("commit %q: %v: %s", msg, err, out)
	}
}

func TestOpenRejectsNonRepo(t *testing.T) {
	if _, err := Open(t.TempDir()); err != ErrNotARepo {
		t.Errorf("err = %v, want ErrNotARepo", err)
	}
}

func TestLatestTagEmptyRepo(t *testing.T) {
	dir := gitInit(t)
	commit(t, dir, "chore: initial")
	r, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	_, ok, err := r.LatestTag()
	if err != nil {
		t.Fatalf("LatestTag: %v", err)
	}
	if ok {
		t.Error("expected no tag in fresh repo")
	}
}

func TestCommitsSinceAndTagFlow(t *testing.T) {
	dir := gitInit(t)
	commit(t, dir, "chore: initial")

	r, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.CreateTag("v0.1.0", "release v0.1.0"); err != nil {
		t.Fatalf("CreateTag: %v", err)
	}

	tag, ok, err := r.LatestTag()
	if err != nil || !ok || tag != "v0.1.0" {
		t.Fatalf("LatestTag = %q,%v,%v", tag, ok, err)
	}

	commit(t, dir, "feat: add facial attendance")
	commit(t, dir, "fix(kiosk): header alignment")

	commits, err := r.CommitsSince("v0.1.0")
	if err != nil {
		t.Fatalf("CommitsSince: %v", err)
	}
	if len(commits) != 2 {
		t.Fatalf("got %d commits, want 2: %+v", len(commits), commits)
	}
	if commits[0].Subject != "feat: add facial attendance" {
		t.Errorf("commits[0] = %q (want oldest first)", commits[0].Subject)
	}

	if has, _ := r.HasTag("v0.1.0"); !has {
		t.Error("HasTag(v0.1.0) = false")
	}
	if has, _ := r.HasTag("v9.9.9"); has {
		t.Error("HasTag(v9.9.9) = true")
	}
}

func TestIsClean(t *testing.T) {
	dir := gitInit(t)
	commit(t, dir, "chore: initial")
	r, _ := Open(dir)
	clean, err := r.IsClean()
	if err != nil || !clean {
		t.Errorf("IsClean = %v,%v want true,nil", clean, err)
	}
}
