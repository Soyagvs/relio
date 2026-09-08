package cmd

import (
	"bytes"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/soyagvs/relio/internal/config"
	"github.com/soyagvs/relio/internal/gitrepo"
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
