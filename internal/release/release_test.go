package release

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/soyagvs/relio/internal/changelog"
	"github.com/soyagvs/relio/internal/config"
	"github.com/soyagvs/relio/internal/conventional"
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

func TestSectionUsesNotesOverride(t *testing.T) {
	p := Plan{
		Next:          semver.Version{Major: 1, Minor: 6, Prefix: "v"},
		Now:           fixedNow,
		Notes:         changelog.Build([]conventional.Commit{{Type: "feat", Description: "generated line"}}),
		NotesOverride: "### Added\n\n- Custom thing",
	}
	got := p.Section()
	if !strings.HasPrefix(got, "## [1.6.0] - ") {
		t.Errorf("Section() lost its heading:\n%s", got)
	}
	if !strings.HasSuffix(got, "### Added\n\n- Custom thing") {
		t.Errorf("Section() did not end with the custom body:\n%s", got)
	}
	if strings.Contains(got, "Generated line") {
		t.Errorf("Section() still carries the generated notes:\n%s", got)
	}
}

func TestReleaseBodyUsesNotesOverride(t *testing.T) {
	p := Plan{
		Next:          semver.Version{Major: 1, Minor: 6, Prefix: "v"},
		Now:           fixedNow,
		Notes:         changelog.Build([]conventional.Commit{{Type: "feat", Description: "generated line"}}),
		NotesOverride: "### Added\n\n- Custom thing",
	}
	if got := p.ReleaseBody(); got != "### Added\n\n- Custom thing" {
		t.Errorf("ReleaseBody() = %q, want the trimmed custom body with no heading", got)
	}
}

func TestSectionIgnoresBlankOverride(t *testing.T) {
	p := Plan{
		Next:          semver.Version{Major: 1, Minor: 6, Prefix: "v"},
		Now:           fixedNow,
		Notes:         changelog.Build([]conventional.Commit{{Type: "feat", Description: "generated line"}}),
		NotesOverride: "   \n",
	}
	got := p.Section()
	if !strings.Contains(got, "Generated line") {
		t.Errorf("blank override should fall back to the rendered Notes:\n%s", got)
	}
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestBuildPlanPopulatesVersionChanges(t *testing.T) {
	dir, r := newRepo(t)
	commit(t, dir, "chore: init")
	tag(t, dir, "v1.5.0")
	writeFile(t, dir, "package.json", `{"name":"x","version":"1.5.0"}`)
	commit(t, dir, "feat: something")

	cfg := config.Default("x")
	cfg.Release.VersionFiles = []config.VersionFile{{Path: "package.json"}}

	p, err := BuildPlan(r, cfg, Options{Now: fixedNow})
	if err != nil {
		t.Fatal(err)
	}
	if !p.VersionFilesUpdate {
		t.Error("VersionFilesUpdate = false, want true")
	}
	if len(p.VersionChanges) != 1 {
		t.Fatalf("VersionChanges = %+v", p.VersionChanges)
	}
	c := p.VersionChanges[0]
	if c.Rel != "package.json" || c.Old != "1.5.0" || c.New != "1.6.0" {
		t.Errorf("change = %+v, want package.json 1.5.0 -> 1.6.0", c)
	}
}

func TestBuildPlanVersionFileErrorFailsEarly(t *testing.T) {
	dir, r := newRepo(t)
	commit(t, dir, "chore: init")
	tag(t, dir, "v1.5.0")
	commit(t, dir, "feat: something")

	cfg := config.Default("x")
	cfg.Release.VersionFiles = []config.VersionFile{{Path: "package.json"}} // never created

	if _, err := BuildPlan(r, cfg, Options{Now: fixedNow}); err == nil ||
		!strings.Contains(err.Error(), "file not found") {
		t.Fatalf("err = %v, want 'file not found'", err)
	}
}

func TestApplyWritesVersionFilesInReleaseCommit(t *testing.T) {
	dir, r := newRepo(t)
	commit(t, dir, "chore: init")
	tag(t, dir, "v1.5.0")
	writeFile(t, dir, "package.json", `{"name":"x","version":"1.5.0"}`)
	if out, err := gitOut(dir, "add", "package.json"); err != nil {
		t.Fatalf("git add: %v: %s", err, out)
	}
	commit(t, dir, "feat: something")

	cfg := config.Default("x")
	cfg.Release.VersionFiles = []config.VersionFile{{Path: "package.json"}}

	p, err := BuildPlan(r, cfg, Options{Now: fixedNow})
	if err != nil {
		t.Fatal(err)
	}
	res, err := p.Apply(r)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}

	if len(res.VersionFiles) != 1 || res.VersionFiles[0] != "package.json" {
		t.Errorf("res.VersionFiles = %+v", res.VersionFiles)
	}
	data, _ := os.ReadFile(filepath.Join(dir, "package.json"))
	if !strings.Contains(string(data), `"version":"1.6.0"`) {
		t.Errorf("package.json not updated:\n%s", data)
	}
	if clean, _ := r.IsClean(); !clean {
		t.Error("work tree should be clean: the version file must be committed")
	}
	stat, err := gitOut(dir, "show", "--stat", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stat, "package.json") || !strings.Contains(stat, "CHANGELOG.md") {
		t.Errorf("release commit does not carry both files:\n%s", stat)
	}
	if subject, _ := gitOut(dir, "log", "-1", "--format=%s"); subject != "chore(release): v1.6.0" {
		t.Errorf("release commit subject = %q", subject)
	}
	tagged, err := gitOut(dir, "show", "v1.6.0:package.json")
	if err != nil || !strings.Contains(tagged, "1.6.0") {
		t.Errorf("tagged package.json missing new version: %v\n%s", err, tagged)
	}
}

func TestApplyVersionFilesDisabledOverride(t *testing.T) {
	dir, r := newRepo(t)
	commit(t, dir, "chore: init")
	tag(t, dir, "v1.5.0")
	writeFile(t, dir, "package.json", `{"name":"x","version":"1.5.0"}`)
	if out, err := gitOut(dir, "add", "package.json"); err != nil {
		t.Fatalf("git add: %v: %s", err, out)
	}
	commit(t, dir, "feat: something")

	cfg := config.Default("x")
	cfg.Release.VersionFiles = []config.VersionFile{{Path: "package.json"}}

	p, err := BuildPlan(r, cfg, Options{Now: fixedNow})
	if err != nil {
		t.Fatal(err)
	}
	p.VersionFilesUpdate = false // what --no-version-files does

	res, err := p.Apply(r)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if len(res.VersionFiles) != 0 {
		t.Errorf("res.VersionFiles = %+v, want none", res.VersionFiles)
	}
	data, _ := os.ReadFile(filepath.Join(dir, "package.json"))
	if !strings.Contains(string(data), `"version":"1.5.0"`) {
		t.Errorf("package.json should be untouched:\n%s", data)
	}
	stat, _ := gitOut(dir, "show", "--stat", "HEAD")
	if strings.Contains(stat, "package.json") {
		t.Errorf("package.json must not be in the release commit:\n%s", stat)
	}
}

func TestPlanLint(t *testing.T) {
	p := Plan{Commits: []conventional.Commit{
		{Type: "feat", Description: "a", Raw: "feat: a"},
		{Raw: "just some wip"},
		{Type: "fix", Scope: "kiosk", Description: "b", Raw: "fix(kiosk): b"},
		{Raw: "another loose one"},
		{Type: "chore", Description: "c", Raw: "chore: c"},
	}}

	conv, nonConv := p.Lint()

	if len(conv) != 3 || len(nonConv) != 2 {
		t.Fatalf("split = %d conventional, %d not; want 3 and 2", len(conv), len(nonConv))
	}
	if conv[0].Type != "feat" || conv[1].Type != "fix" || conv[2].Type != "chore" {
		t.Errorf("conventional order not preserved: %+v", conv)
	}
	if nonConv[0].Raw != "just some wip" || nonConv[1].Raw != "another loose one" {
		t.Errorf("non-conventional order not preserved: %+v", nonConv)
	}
}

func TestPlanLintAllConventional(t *testing.T) {
	p := Plan{Commits: []conventional.Commit{
		{Type: "feat", Raw: "feat: a"},
		{Type: "fix", Raw: "fix: b"},
	}}
	conv, nonConv := p.Lint()
	if len(conv) != 2 {
		t.Errorf("conv = %d, want 2", len(conv))
	}
	if len(nonConv) != 0 {
		t.Errorf("nonConv = %d, want 0", len(nonConv))
	}
}

// --- pre-release / release-candidate flow -----------------------------------

func TestBuildPlanRCFromStable(t *testing.T) {
	// On stable v1.5.0 with a feat since -> `relio --rc` cuts v1.6.0-rc.1.
	dir, r := newRepo(t)
	commit(t, dir, "chore: init")
	tag(t, dir, "v1.5.0")
	commit(t, dir, "feat: something")

	p, err := BuildPlan(r, config.Default("x"), Options{Now: fixedNow, Prerelease: true})
	if err != nil {
		t.Fatal(err)
	}
	if p.Next.String() != "v1.6.0-rc.1" {
		t.Errorf("Next = %s, want v1.6.0-rc.1", p.Next)
	}
	if !p.Prerelease || p.Finalizing {
		t.Errorf("flags: Prerelease=%v Finalizing=%v", p.Prerelease, p.Finalizing)
	}
	if p.NothingToRelease() {
		t.Error("NothingToRelease() = true")
	}
}

func TestBuildPlanRCBumpsCounter(t *testing.T) {
	// On v1.6.0-rc.1 with only a fix since -> `relio --rc` cuts v1.6.0-rc.2,
	// NOT v1.5.1-rc.1 (Compare guards the core against the current rc's core).
	dir, r := newRepo(t)
	commit(t, dir, "chore: init")
	tag(t, dir, "v1.5.0")
	commit(t, dir, "feat: something")
	tag(t, dir, "v1.6.0-rc.1")
	commit(t, dir, "fix: a bug")

	p, err := BuildPlan(r, config.Default("x"), Options{Now: fixedNow, Prerelease: true})
	if err != nil {
		t.Fatal(err)
	}
	if p.Next.String() != "v1.6.0-rc.2" {
		t.Errorf("Next = %s, want v1.6.0-rc.2", p.Next)
	}
	if p.Current.String() != "v1.6.0-rc.1" {
		t.Errorf("Current = %s, want v1.6.0-rc.1", p.Current)
	}
}

func TestBuildPlanRCEscalatesCore(t *testing.T) {
	// On v1.6.0-rc.1 with a feat! since -> `relio --rc` moves the core up and
	// restarts the counter -> v2.0.0-rc.1.
	dir, r := newRepo(t)
	commit(t, dir, "chore: init")
	tag(t, dir, "v1.5.0")
	commit(t, dir, "feat: something")
	tag(t, dir, "v1.6.0-rc.1")
	commit(t, dir, "feat!: overhaul the api")

	p, err := BuildPlan(r, config.Default("x"), Options{Now: fixedNow, Prerelease: true})
	if err != nil {
		t.Fatal(err)
	}
	if p.Next.String() != "v2.0.0-rc.1" {
		t.Errorf("Next = %s, want v2.0.0-rc.1", p.Next)
	}
}

func TestBuildPlanFinalizeRC(t *testing.T) {
	// On v1.6.0-rc.2, `relio` (no flag) with 0 new commits finalizes v1.6.0.
	// The commit range starts at the stable base, so the section summarises the
	// whole span: 2 commits made before rc.1.
	dir, r := newRepo(t)
	commit(t, dir, "chore: init")
	tag(t, dir, "v1.5.0")
	commit(t, dir, "feat: one")
	commit(t, dir, "fix: two")
	tag(t, dir, "v1.6.0-rc.1")
	tag(t, dir, "v1.6.0-rc.2")

	p, err := BuildPlan(r, config.Default("x"), Options{Now: fixedNow})
	if err != nil {
		t.Fatal(err)
	}
	if p.Next.String() != "v1.6.0" {
		t.Errorf("Next = %s, want v1.6.0", p.Next)
	}
	if !p.Finalizing || p.Prerelease {
		t.Errorf("flags: Finalizing=%v Prerelease=%v", p.Finalizing, p.Prerelease)
	}
	if p.NothingToRelease() {
		t.Error("NothingToRelease() = true, want false when finalizing")
	}
	if len(p.Commits) != 2 {
		t.Errorf("Commits = %d, want 2 (whole span since the stable base)", len(p.Commits))
	}
}

func TestBuildPlanRCNoTagsFirstRC(t *testing.T) {
	// No tags + `--rc` + a feat -> v0.1.0-rc.1.
	dir, r := newRepo(t)
	commit(t, dir, "feat: first")

	p, err := BuildPlan(r, config.Default("x"), Options{Now: fixedNow, Prerelease: true})
	if err != nil {
		t.Fatal(err)
	}
	if p.Next.String() != "v0.1.0-rc.1" {
		t.Errorf("Next = %s, want v0.1.0-rc.1", p.Next)
	}
}

func TestBuildPlanRCNothingToRelease(t *testing.T) {
	// Stable v1.5.0 + 0 commits + `--rc` -> NothingToRelease() stays true.
	dir, r := newRepo(t)
	commit(t, dir, "chore: init")
	tag(t, dir, "v1.5.0")

	p, err := BuildPlan(r, config.Default("x"), Options{Now: fixedNow, Prerelease: true})
	if err != nil {
		t.Fatal(err)
	}
	if !p.NothingToRelease() {
		t.Error("NothingToRelease() = false, want true")
	}
}

func TestTagNamePrerelease(t *testing.T) {
	p := Plan{
		Config: config.Default("x"),
		Next:   semver.Version{Major: 1, Minor: 6, Prefix: "v", Pre: "rc.1"},
	}
	if got := p.TagName(); got != "v1.6.0-rc.1" {
		t.Errorf("TagName() = %q, want v1.6.0-rc.1", got)
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
	disabled := false
	cfg.Release.Changelog = &disabled // isolate the tag path
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
