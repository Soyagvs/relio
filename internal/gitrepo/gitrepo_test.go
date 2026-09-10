package gitrepo

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
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

func commitAs(t *testing.T, dir, name, email, msg string) {
	t.Helper()
	cmd := exec.Command("git",
		"-c", "user.name="+name, "-c", "user.email="+email,
		"commit", "--allow-empty", "-q", "-m", msg)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("commit as %q: %v: %s", name, err, out)
	}
}

func TestAuthorsBetween(t *testing.T) {
	dir := gitInit(t)
	commit(t, dir, "chore: initial")
	r, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.CreateTag("v1.0.0", "release v1.0.0"); err != nil {
		t.Fatal(err)
	}

	commitAs(t, dir, "Alice", "alice@example.com", "feat: one")
	commitAs(t, dir, "Bob", "bob@example.com", "fix: two")
	commitAs(t, dir, "Alice", "alice@example.com", "feat: three") // repeat
	commitAs(t, dir, "release-bot[bot]", "bot@example.com", "chore: bump")

	got, err := r.AuthorsBetween("v1.0.0", "")
	if err != nil {
		t.Fatalf("AuthorsBetween: %v", err)
	}
	want := []string{"Alice", "Bob"}
	if len(got) != len(want) {
		t.Fatalf("AuthorsBetween = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("AuthorsBetween[%d] = %q, want %q (full: %v)", i, got[i], want[i], got)
		}
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

func TestLatestStableTag(t *testing.T) {
	dir := gitInit(t)
	commit(t, dir, "chore: initial")
	r, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}

	// No tags at all.
	if _, ok, err := r.LatestStableTag(); err != nil || ok {
		t.Fatalf("empty repo: tag,ok,err = _,%v,%v", ok, err)
	}

	// Only a pre-release tag.
	if err := r.CreateTag("v0.1.0-rc.1", "rc"); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := r.LatestStableTag(); err != nil || ok {
		t.Fatalf("rc-only repo: tag,ok,err = _,%v,%v", ok, err)
	}

	// Stable tags mixed with a newer pre-release: the newest stable wins.
	if err := r.CreateTag("v1.0.0", "release"); err != nil {
		t.Fatal(err)
	}
	if err := r.CreateTag("v1.1.0", "release"); err != nil {
		t.Fatal(err)
	}
	if err := r.CreateTag("v1.2.0-rc.1", "rc"); err != nil {
		t.Fatal(err)
	}
	tag, ok, err := r.LatestStableTag()
	if err != nil || !ok || tag != "v1.1.0" {
		t.Fatalf("LatestStableTag = %q,%v,%v want v1.1.0,true,nil", tag, ok, err)
	}
}

func TestTagsAndDeleteTag(t *testing.T) {
	dir := gitInit(t)
	commit(t, dir, "chore: initial")
	r, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}

	if tags, _ := r.Tags(); len(tags) != 0 {
		t.Fatalf("fresh repo has tags: %+v", tags)
	}

	if err := r.CreateTag("v0.1.0", "release v0.1.0"); err != nil {
		t.Fatal(err)
	}
	commit(t, dir, "feat: more")
	if err := r.CreateTag("v0.2.0", "release v0.2.0"); err != nil {
		t.Fatal(err)
	}
	commit(t, dir, "feat: even more")
	if err := r.CreateTag("v0.10.0", "release v0.10.0"); err != nil {
		t.Fatal(err)
	}

	tags, err := r.Tags()
	if err != nil {
		t.Fatal(err)
	}
	if len(tags) != 3 {
		t.Fatalf("want 3 tags, got %d: %+v", len(tags), tags)
	}
	// version sort, newest first: v0.10.0 > v0.2.0 > v0.1.0
	if tags[0].Name != "v0.10.0" || tags[1].Name != "v0.2.0" || tags[2].Name != "v0.1.0" {
		t.Errorf("bad order: %+v", tags)
	}
	if tags[0].Subject != "release v0.10.0" {
		t.Errorf("subject = %q", tags[0].Subject)
	}
	if tags[0].Date == "" {
		t.Errorf("date is empty")
	}
	if m, _ := regexp.MatchString(`^\d{4}-\d{2}-\d{2} \d{2}:\d{2}$`, tags[0].DateTime); !m {
		t.Errorf("DateTime = %q, want YYYY-MM-DD HH:MM", tags[0].DateTime)
	}

	msg, err := r.TagMessage("v0.2.0")
	if err != nil || !strings.Contains(msg, "release v0.2.0") {
		t.Errorf("TagMessage = %q, %v", msg, err)
	}

	if err := r.DeleteTag("v0.2.0"); err != nil {
		t.Fatalf("DeleteTag: %v", err)
	}
	if has, _ := r.HasTag("v0.2.0"); has {
		t.Error("v0.2.0 still present after delete")
	}
	if tags, _ := r.Tags(); len(tags) != 2 {
		t.Errorf("want 2 tags after delete, got %d", len(tags))
	}
}

func TestHeadSubject(t *testing.T) {
	dir := gitInit(t)
	commit(t, dir, "chore(release): v1.2.0")
	r, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	got, err := r.HeadSubject()
	if err != nil {
		t.Fatalf("HeadSubject: %v", err)
	}
	if got != "chore(release): v1.2.0" {
		t.Errorf("HeadSubject = %q, want %q", got, "chore(release): v1.2.0")
	}
}

func TestTagPointsAtHead(t *testing.T) {
	dir := gitInit(t)
	commit(t, dir, "chore: initial")
	r, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.CreateTag("v1.0.0", "release v1.0.0"); err != nil {
		t.Fatal(err)
	}
	at, err := r.TagPointsAtHead("v1.0.0")
	if err != nil {
		t.Fatalf("TagPointsAtHead: %v", err)
	}
	if !at {
		t.Error("TagPointsAtHead = false right after tagging HEAD")
	}

	commit(t, dir, "feat: move on")
	at, err = r.TagPointsAtHead("v1.0.0")
	if err != nil {
		t.Fatalf("TagPointsAtHead: %v", err)
	}
	if at {
		t.Error("TagPointsAtHead = true after a later commit")
	}
}

func TestRemoteContainsHead(t *testing.T) {
	dir := gitInit(t)
	commit(t, dir, "chore: initial")
	r, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}

	has, err := r.RemoteContainsHead()
	if err != nil {
		t.Fatalf("RemoteContainsHead: %v", err)
	}
	if has {
		t.Error("RemoteContainsHead = true with no remotes")
	}

	bare := t.TempDir()
	if out, err := exec.Command("git", "init", "--bare", "-q", bare).CombinedOutput(); err != nil {
		t.Fatalf("git init --bare: %v: %s", err, out)
	}
	if _, err := run(dir, "remote", "add", "origin", bare); err != nil {
		t.Fatalf("remote add: %v", err)
	}
	branch, err := r.CurrentBranch()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := run(dir, "push", "-u", "origin", branch); err != nil {
		t.Fatalf("push: %v", err)
	}

	has, err = r.RemoteContainsHead()
	if err != nil {
		t.Fatalf("RemoteContainsHead: %v", err)
	}
	if !has {
		t.Error("RemoteContainsHead = false after pushing HEAD")
	}
}

func TestResetHardPrevious(t *testing.T) {
	dir := gitInit(t)
	r, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}

	write := func(name, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := run(dir, "add", name); err != nil {
			t.Fatal(err)
		}
	}
	write("a.txt", "one\n")
	if _, err := run(dir, "commit", "-q", "-m", "feat: first"); err != nil {
		t.Fatal(err)
	}
	write("b.txt", "two\n")
	if _, err := run(dir, "commit", "-q", "-m", "chore(release): v1.0.0"); err != nil {
		t.Fatal(err)
	}

	if err := r.ResetHardPrevious(); err != nil {
		t.Fatalf("ResetHardPrevious: %v", err)
	}
	subject, err := r.HeadSubject()
	if err != nil {
		t.Fatal(err)
	}
	if subject != "feat: first" {
		t.Errorf("HeadSubject = %q, want %q", subject, "feat: first")
	}
	if clean, _ := r.IsClean(); !clean {
		t.Error("IsClean = false after ResetHardPrevious")
	}
	if _, err := os.Stat(filepath.Join(dir, "b.txt")); !os.IsNotExist(err) {
		t.Errorf("b.txt still present after reset: %v", err)
	}
}

func TestCurrentBranch(t *testing.T) {
	dir := gitInit(t)
	commit(t, dir, "chore: initial")
	r, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	br, err := r.CurrentBranch()
	if err != nil {
		t.Fatalf("CurrentBranch: %v", err)
	}
	if br != "main" && br != "master" {
		t.Errorf("CurrentBranch = %q, want main or master", br)
	}
}

func TestPush(t *testing.T) {
	dir := gitInit(t)
	commit(t, dir, "chore: initial")
	r, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}

	bare := t.TempDir()
	if out, err := exec.Command("git", "init", "--bare", "-q", bare).CombinedOutput(); err != nil {
		t.Fatalf("git init --bare: %v: %s", err, out)
	}
	if _, err := run(dir, "remote", "add", "origin", bare); err != nil {
		t.Fatalf("remote add: %v", err)
	}

	branch, err := r.CurrentBranch()
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Push("origin", branch); err != nil {
		t.Fatalf("Push branch: %v", err)
	}
	if out, err := exec.Command("git", "--git-dir", bare, "rev-parse", branch).CombinedOutput(); err != nil {
		t.Fatalf("bare repo has no %s after push: %v: %s", branch, err, out)
	}

	if err := r.CreateTag("v0.1.0", "release v0.1.0"); err != nil {
		t.Fatal(err)
	}
	if err := r.Push("origin", "v0.1.0"); err != nil {
		t.Fatalf("Push tag: %v", err)
	}
	if out, err := exec.Command("git", "--git-dir", bare, "rev-parse", "refs/tags/v0.1.0").CombinedOutput(); err != nil {
		t.Fatalf("bare repo has no tag after push: %v: %s", err, out)
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

func TestCommitPathsCommitsOnlyGivenFiles(t *testing.T) {
	dir := gitInit(t)
	commit(t, dir, "chore: initial")
	r, _ := Open(dir)

	write := func(name, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("CHANGELOG.md", "# Changelog\n")
	write("other.txt", "untracked, must stay out of the release commit\n")

	if err := r.CommitPaths("chore(release): v1.0.0", "CHANGELOG.md"); err != nil {
		t.Fatalf("CommitPaths: %v", err)
	}

	subject, err := run(dir, "log", "-1", "--format=%s")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(subject) != "chore(release): v1.0.0" {
		t.Errorf("subject = %q", strings.TrimSpace(subject))
	}
	files, err := run(dir, "show", "--name-only", "--format=", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(files); got != "CHANGELOG.md" {
		t.Errorf("committed files = %q, want just CHANGELOG.md", got)
	}
	if clean, _ := r.IsClean(); clean {
		t.Error("other.txt should still be uncommitted")
	}
}
