package cmd

import (
	"bytes"
	"io"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/soyagvs/relio/internal/config"
	"github.com/soyagvs/relio/internal/gitrepo"
	"github.com/soyagvs/relio/internal/i18n"
)

func initTempRepo(t *testing.T) (*gitrepo.Repo, string) {
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
	return r, dir
}

func TestRunInitWritesConfig(t *testing.T) {
	repo, dir := initTempRepo(t)

	var buf bytes.Buffer
	if err := runInit(&buf, repo, "widget"); err != nil {
		t.Fatalf("runInit: %v", err)
	}
	if !config.Exists(dir) {
		t.Fatal("runInit did not create .release.yaml")
	}
	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.Project != "widget" {
		t.Errorf("project = %q, want %q (the override)", cfg.Project, "widget")
	}
	if !strings.Contains(buf.String(), filepath.Join(dir, config.FileName)) {
		t.Errorf("output missing the created path:\n%s", buf.String())
	}
}

func TestRunInitRefusesExistingConfig(t *testing.T) {
	repo, dir := initTempRepo(t)
	if err := config.Default("x").Save(dir); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	err := runInit(&buf, repo, "")
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Errorf("runInit on an existing config = %v, want an 'already exists' error", err)
	}
}

// --- i18n: construction-time Short/Long/flag usage + runtime output localization ---

func TestNewInitCmdShortLongFlagLocalizeAtConstructionTime(t *testing.T) {
	prev := i18n.Current()
	t.Cleanup(func() { i18n.SetLanguage(prev) })

	i18n.SetLanguage("en")
	cEN := newInitCmd(&releaseFlags{})
	shortEN, longEN := cEN.Short, cEN.Long
	usageEN := cEN.Flags().Lookup("project").Usage

	i18n.SetLanguage("es")
	cES := newInitCmd(&releaseFlags{})
	shortES, longES := cES.Short, cES.Long
	usageES := cES.Flags().Lookup("project").Usage

	if shortEN != "Create a .release.yaml in the current repository" {
		t.Errorf("newInitCmd().Short (en) = %q", shortEN)
	}
	if longEN != "Write a .release.yaml with sensible defaults. The file holds configuration only — never secrets." {
		t.Errorf("newInitCmd().Long (en) = %q", longEN)
	}
	if shortES == shortEN || shortES == "" {
		t.Errorf("newInitCmd().Short unchanged across languages: %q", shortES)
	}
	if longES == longEN || longES == "" {
		t.Errorf("newInitCmd().Long unchanged across languages: %q", longES)
	}
	if usageES == usageEN || usageES == "" {
		t.Errorf("--project usage unchanged across languages: %q", usageES)
	}
}

func TestRunInitLocalizesOutput(t *testing.T) {
	repoEN, dirEN := initTempRepo(t)
	repoES, dirES := initTempRepo(t)

	prev := i18n.Current()
	t.Cleanup(func() { i18n.SetLanguage(prev) })

	i18n.SetLanguage("en")
	var bufEN bytes.Buffer
	if err := runInit(&bufEN, repoEN, "widget"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(bufEN.String(), "created") {
		t.Errorf("english golden text missing:\n%s", bufEN.String())
	}
	if !strings.Contains(bufEN.String(), "Review it, commit it, then run `relio`.") {
		t.Errorf("english golden hint missing:\n%s", bufEN.String())
	}

	i18n.SetLanguage("es")
	var bufES bytes.Buffer
	if err := runInit(&bufES, repoES, "widget"); err != nil {
		t.Fatal(err)
	}

	// Normalize the two distinct temp-repo paths before comparing so the
	// assertion is about localized text, not incidental path differences.
	normEN := strings.ReplaceAll(bufEN.String(), dirEN, "<root>")
	normES := strings.ReplaceAll(bufES.String(), dirES, "<root>")
	if normEN == normES {
		t.Error("runInit output unchanged across languages")
	}
}

func TestRunInitRefusesExistingConfigLocalizesError(t *testing.T) {
	repo, dir := initTempRepo(t)
	if err := config.Default("x").Save(dir); err != nil {
		t.Fatal(err)
	}

	prev := i18n.Current()
	t.Cleanup(func() { i18n.SetLanguage(prev) })

	i18n.SetLanguage("en")
	errEN := runInit(io.Discard, repo, "")
	i18n.SetLanguage("es")
	errES := runInit(io.Discard, repo, "")

	if errEN == nil || errES == nil {
		t.Fatal("expected already-exists errors in both languages")
	}
	if !strings.Contains(errEN.Error(), "already exists") {
		t.Errorf("english golden error text missing: %q", errEN.Error())
	}
	if errEN.Error() == errES.Error() {
		t.Errorf("already-exists error unchanged across languages: %q", errEN.Error())
	}
}

func TestNewInitCmdNotAGitRepoErrorLocalizes(t *testing.T) {
	dir := t.TempDir()
	f := &releaseFlags{dir: dir}

	prev := i18n.Current()
	t.Cleanup(func() { i18n.SetLanguage(prev) })

	i18n.SetLanguage("en")
	cEN := newInitCmd(f)
	errEN := cEN.RunE(cEN, nil)
	i18n.SetLanguage("es")
	cES := newInitCmd(f)
	errES := cES.RunE(cES, nil)

	if errEN == nil || errES == nil {
		t.Fatal("expected not-a-git-repo errors in both languages")
	}
	if !strings.Contains(errEN.Error(), "not a git repository") {
		t.Errorf("english golden error text missing: %q", errEN.Error())
	}
	if errEN.Error() == errES.Error() {
		t.Errorf("not-a-git-repo error unchanged across languages: %q", errEN.Error())
	}
}
