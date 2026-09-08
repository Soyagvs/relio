package release

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/soyagvs/relio/internal/config"
	"github.com/soyagvs/relio/internal/gitrepo"
	"github.com/soyagvs/relio/internal/semver"
)

func newRepo(t *testing.T) (string, *gitrepo.Repo) {
	t.Helper()
	dir := t.TempDir()
	for _, args := range [][]string{
		{"init", "-q"},
		{"config", "user.email", "t@e.com"},
		{"config", "user.name", "T"},
		{"config", "commit.gpgsign", "false"},
	} {
		c := exec.Command("git", args...)
		c.Dir = dir
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	r, err := gitrepo.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	return dir, r
}

func commit(t *testing.T, dir, msg string) {
	t.Helper()
	c := exec.Command("git", "commit", "--allow-empty", "-q", "-m", msg)
	c.Dir = dir
	if out, err := c.CombinedOutput(); err != nil {
		t.Fatalf("commit: %v: %s", err, out)
	}
}

func tag(t *testing.T, dir, name string) {
	t.Helper()
	c := exec.Command("git", "tag", name)
	c.Dir = dir
	if out, err := c.CombinedOutput(); err != nil {
		t.Fatalf("tag: %v: %s", err, out)
	}
}

var fixedNow = time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC)

func TestBuildPlanFirstRelease(t *testing.T) {
	dir, r := newRepo(t)
	commit(t, dir, "feat: add facial attendance")
	commit(t, dir, "fix: kiosk header")

	p, err := BuildPlan(r, config.Default("azeink"), Options{Now: fixedNow})
	if err != nil {
		t.Fatal(err)
	}
	if p.Current.String() != "v0.0.0" {
		t.Errorf("Current = %s", p.Current)
	}
	if p.Bump != semver.Minor {
		t.Errorf("Bump = %v, want minor", p.Bump)
	}
	if p.Next.String() != "v0.1.0" {
		t.Errorf("Next = %s", p.Next)
	}
	if len(p.Commits) != 2 {
		t.Errorf("Commits = %d", len(p.Commits))
	}
}

func TestBuildPlanSinceTag(t *testing.T) {
	dir, r := newRepo(t)
	commit(t, dir, "chore: init")
	tag(t, dir, "v1.3.2")
	commit(t, dir, "feat: add categories")
	commit(t, dir, "fix: dashboard")
	commit(t, dir, "refactor: auth flow")

	p, err := BuildPlan(r, config.Default("azeink"), Options{Now: fixedNow})
	if err != nil {
		t.Fatal(err)
	}
	if p.Current.String() != "v1.3.2" || p.Next.String() != "v1.4.0" {
		t.Errorf("versions = %s -> %s", p.Current, p.Next)
	}
	if len(p.Commits) != 3 {
		t.Errorf("Commits = %d, want 3", len(p.Commits))
	}
}

func TestBuildPlanForceMajor(t *testing.T) {
	dir, r := newRepo(t)
	commit(t, dir, "chore: init")
	tag(t, dir, "v1.3.2")
	commit(t, dir, "fix: small")

	p, err := BuildPlan(r, config.Default("x"), Options{Now: fixedNow, ForceBump: semver.Major})
	if err != nil {
		t.Fatal(err)
	}
	if !p.BumpForced || p.Next.String() != "v2.0.0" {
		t.Errorf("forced major failed: forced=%v next=%s", p.BumpForced, p.Next)
	}
}

func TestBuildPlanNoConventionalSignalDefaultsPatch(t *testing.T) {
	dir, r := newRepo(t)
	commit(t, dir, "chore: init")
	tag(t, dir, "v1.0.0")
	commit(t, dir, "updated some docs")

	p, err := BuildPlan(r, config.Default("x"), Options{Now: fixedNow})
	if err != nil {
		t.Fatal(err)
	}
	if p.Bump != semver.Patch || p.Next.String() != "v1.0.1" {
		t.Errorf("default patch failed: %v %s", p.Bump, p.Next)
	}
}

func TestApplyWritesChangelogAndTag(t *testing.T) {
	dir, r := newRepo(t)
	commit(t, dir, "chore: init")
	tag(t, dir, "v1.3.2")
	commit(t, dir, "feat: add categories")
	commit(t, dir, "fix: dashboard crash")

	p, err := BuildPlan(r, config.Default("azeink"), Options{Now: fixedNow})
	if err != nil {
		t.Fatal(err)
	}
	res, err := p.Apply(r)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}

	if res.TagName != "v1.4.0" {
		t.Errorf("TagName = %q", res.TagName)
	}
	if has, _ := r.HasTag("v1.4.0"); !has {
		t.Error("tag not created")
	}

	data, err := os.ReadFile(filepath.Join(dir, "CHANGELOG.md"))
	if err != nil {
		t.Fatal(err)
	}
	cl := string(data)
	if !strings.HasPrefix(cl, "# Changelog") {
		t.Errorf("missing header:\n%s", cl)
	}
	if !strings.Contains(cl, "## [1.4.0] - 2026-09-06") {
		t.Errorf("missing section:\n%s", cl)
	}
	if !strings.Contains(cl, "Add categories") || !strings.Contains(cl, "Dashboard crash") {
		t.Errorf("missing notes:\n%s", cl)
	}
}

func TestApplyCommitsChangelogBeforeTag(t *testing.T) {
	dir, r := newRepo(t)
	commit(t, dir, "chore: init")
	tag(t, dir, "v1.0.0")
	commit(t, dir, "feat: add categories")

	p, err := BuildPlan(r, config.Default("x"), Options{Now: fixedNow})
	if err != nil {
		t.Fatal(err)
	}
	res, err := p.Apply(r)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}

	if !res.Committed {
		t.Error("res.Committed = false, want true")
	}
	if clean, _ := r.IsClean(); !clean {
		t.Error("work tree should be clean: the changelog must be committed")
	}

	subject, err := gitOut(dir, "log", "-1", "--format=%s")
	if err != nil {
		t.Fatal(err)
	}
	if subject != "chore(release): v1.1.0" {
		t.Errorf("release commit subject = %q", subject)
	}

	// The tag must point at the commit that already carries the changelog entry.
	tagged, err := gitOut(dir, "show", "v1.1.0:CHANGELOG.md")
	if err != nil {
		t.Fatalf("reading CHANGELOG.md at the tag: %v", err)
	}
	if !strings.Contains(tagged, "## [1.1.0] - 2026-09-06") {
		t.Errorf("tagged CHANGELOG.md missing the 1.1.0 section:\n%s", tagged)
	}
}

func TestBuildPlanCarriesPublishIntent(t *testing.T) {
	dir, r := newRepo(t)
	commit(t, dir, "feat: x")

	cfg := config.Default("x")
	cfg.GitHub.Release = true
	p, err := BuildPlan(r, cfg, Options{Now: fixedNow})
	if err != nil {
		t.Fatal(err)
	}
	if !p.PublishGitHub {
		t.Error("PublishGitHub = false, want true from cfg.GitHub.Release")
	}

	cfg.GitHub.Release = false
	p, err = BuildPlan(r, cfg, Options{Now: fixedNow})
	if err != nil {
		t.Fatal(err)
	}
	if p.PublishGitHub {
		t.Error("PublishGitHub = true, want false")
	}
}

func TestReleaseBodyStripsHeading(t *testing.T) {
	dir, r := newRepo(t)
	commit(t, dir, "chore: init")
	tag(t, dir, "v1.0.0")
	commit(t, dir, "feat: add categories")
	commit(t, dir, "fix: dashboard crash")

	p, err := BuildPlan(r, config.Default("x"), Options{Now: fixedNow})
	if err != nil {
		t.Fatal(err)
	}

	body := p.ReleaseBody()
	if strings.Contains(body, "## [1.1.0]") || strings.HasPrefix(body, "## [") {
		t.Errorf("body still carries the heading line:\n%s", body)
	}
	if !strings.HasPrefix(body, "### Added") {
		t.Errorf("body should start at the first section:\n%s", body)
	}
	if !strings.Contains(body, "Add categories") || !strings.Contains(body, "Dashboard crash") {
		t.Errorf("body missing notes:\n%s", body)
	}
	// The full section still keeps its heading — ReleaseBody must not mutate it.
	if !strings.Contains(p.Section(), "## [1.1.0] - 2026-09-06") {
		t.Errorf("Section() lost its heading: %s", p.Section())
	}
}

func gitOut(dir string, args ...string) (string, error) {
	c := exec.Command("git", args...)
	c.Dir = dir
	out, err := c.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

func TestApplyRefusesDuplicateTag(t *testing.T) {
	dir, r := newRepo(t)
	commit(t, dir, "chore: init")
	tag(t, dir, "v1.3.2")
	commit(t, dir, "feat: thing")

	cfg := config.Default("x")
	cfg.Release.Changelog = false // isolate the tag path
	p, err := BuildPlan(r, cfg, Options{Now: fixedNow})
	if err != nil {
		t.Fatal(err)
	}
	tag(t, dir, "v1.4.0") // pre-create the target

	if _, err := p.Apply(r); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Errorf("err = %v, want 'already exists'", err)
	}
}

func TestApplySecondReleasePrependsSection(t *testing.T) {
	dir, r := newRepo(t)
	commit(t, dir, "chore: init")
	tag(t, dir, "v1.0.0")
	commit(t, dir, "feat: first")

	p1, _ := BuildPlan(r, config.Default("x"), Options{Now: fixedNow})
	if _, err := p1.Apply(r); err != nil {
		t.Fatal(err)
	}

	commit(t, dir, "fix: second")
	p2, _ := BuildPlan(r, config.Default("x"), Options{Now: fixedNow.Add(24 * time.Hour)})
	if _, err := p2.Apply(r); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(filepath.Join(dir, "CHANGELOG.md"))
	cl := string(data)
	i110 := strings.Index(cl, "## [1.1.0]")
	i101 := strings.Index(cl, "## [1.0.1]") // wait: p2 bump is patch? first=feat -> 1.1.0, second=fix -> 1.1.1
	_ = i101
	iNewest := strings.Index(cl, "## [1.1.1]")
	if i110 < 0 || iNewest < 0 {
		t.Fatalf("sections missing:\n%s", cl)
	}
	if iNewest > i110 {
		t.Errorf("newest section not on top:\n%s", cl)
	}
}
