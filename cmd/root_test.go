package cmd

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/soyagvs/relio/internal/config"
	"github.com/soyagvs/relio/internal/gitrepo"
	"github.com/soyagvs/relio/internal/semver"
)

// repoWithPendingRelease builds a temp git repo tagged v1.5.0 with one extra
// `feat:` commit on top, so a plain `doRelease` run targets v1.6.0.
func repoWithPendingRelease(t *testing.T) (*gitrepo.Repo, string) {
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
	mk := func(msg string) {
		c := exec.Command("git", "commit", "--allow-empty", "-q", "-m", msg)
		c.Dir = dir
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("commit %q: %v: %s", msg, err, out)
		}
	}
	mk("chore: init")
	tagCmd := exec.Command("git", "tag", "v1.5.0")
	tagCmd.Dir = dir
	if out, err := tagCmd.CombinedOutput(); err != nil {
		t.Fatalf("tag: %v: %s", err, out)
	}
	mk("feat: something big")

	r, err := gitrepo.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	return r, dir
}

func TestRootHasNoHooksFlag(t *testing.T) {
	c := NewRootCmd()
	f := c.Flags().Lookup("no-hooks")
	if f == nil {
		t.Fatal("missing --no-hooks flag on the root command")
	}
	if !strings.Contains(f.Usage, "hooks") {
		t.Errorf("--no-hooks usage = %q", f.Usage)
	}
}

func TestDoReleaseBeforeHookAbort(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("hook assertions use the `sh` shell")
	}
	r, dir := repoWithPendingRelease(t)

	cfg := config.Default("proj")
	cfg.Release.Hooks.Before = config.StringList{"exit 1"}

	var buf bytes.Buffer
	f := &releaseFlags{dir: dir, yes: true}
	err := doRelease(&buf, r, cfg, f, semver.None, false)
	if err == nil {
		t.Fatalf("expected an error, got nil\noutput:\n%s", buf.String())
	}
	if !strings.Contains(err.Error(), "before hook failed") {
		t.Errorf("error = %q, want it to mention 'before hook failed'", err)
	}
	if has, _ := r.HasTag("v1.6.0"); has {
		t.Error("tag v1.6.0 was created despite the before hook aborting")
	}
	if _, serr := os.Stat(filepath.Join(dir, "CHANGELOG.md")); serr == nil {
		t.Error("CHANGELOG.md was written despite the before hook aborting")
	}
	log := exec.Command("git", "log", "--oneline")
	log.Dir = dir
	out, _ := log.CombinedOutput()
	if strings.Contains(string(out), "chore(release)") {
		t.Errorf("a release commit was made despite the abort:\n%s", out)
	}
}

func TestDoReleaseAfterHookNonFatal(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("hook assertions use the `sh` shell")
	}
	r, dir := repoWithPendingRelease(t)

	cfg := config.Default("proj")
	cfg.Release.Hooks.After = config.StringList{"exit 1"}

	var buf bytes.Buffer
	f := &releaseFlags{dir: dir, yes: true}
	if err := doRelease(&buf, r, cfg, f, semver.None, false); err != nil {
		t.Fatalf("doRelease returned %v, want nil (after hook failures only warn)\noutput:\n%s", err, buf.String())
	}
	if has, _ := r.HasTag("v1.6.0"); !has {
		t.Errorf("tag v1.6.0 was not created\noutput:\n%s", buf.String())
	}
	if !strings.Contains(buf.String(), "after hook failed") {
		t.Errorf("missing 'after hook failed' warning:\n%s", buf.String())
	}
}

func TestDoReleaseNoHooksFlagSkips(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("hook assertions use the `sh` shell")
	}
	r, dir := repoWithPendingRelease(t)

	cfg := config.Default("proj")
	cfg.Release.Hooks.Before = config.StringList{"exit 1"}

	var buf bytes.Buffer
	f := &releaseFlags{dir: dir, yes: true, noHooks: true}
	if err := doRelease(&buf, r, cfg, f, semver.None, false); err != nil {
		t.Fatalf("doRelease returned %v, want nil (--no-hooks should skip the failing before hook)", err)
	}
	if has, _ := r.HasTag("v1.6.0"); !has {
		t.Error("tag v1.6.0 was not created with --no-hooks")
	}
}

func TestRootHasRCFlag(t *testing.T) {
	c := NewRootCmd()
	f := c.Flags().Lookup("rc")
	if f == nil {
		t.Fatal("missing --rc flag on the root command")
	}
	if !strings.Contains(f.Usage, "release candidate") {
		t.Errorf("--rc usage = %q", f.Usage)
	}
}

func TestRootHasEditFlag(t *testing.T) {
	c := NewRootCmd()
	f := c.Flags().Lookup("edit")
	if f == nil {
		t.Fatal("missing --edit flag on the root command")
	}
	if !strings.Contains(f.Usage, "editor") {
		t.Errorf("--edit usage = %q", f.Usage)
	}
}

func TestDoReleaseEditNeedsInteractiveTerminal(t *testing.T) {
	r, dir := repoWithPendingRelease(t)

	var buf bytes.Buffer
	f := &releaseFlags{dir: dir, yes: true, edit: true}
	err := doRelease(&buf, r, config.Default("proj"), f, semver.None, false)
	if err == nil || !strings.Contains(err.Error(), "--edit needs an interactive terminal") {
		t.Fatalf("err = %v, want '--edit needs an interactive terminal'", err)
	}
	if has, _ := r.HasTag("v1.6.0"); has {
		t.Error("tag v1.6.0 created despite the --edit precondition failing")
	}
	if _, serr := os.Stat(filepath.Join(dir, "CHANGELOG.md")); serr == nil {
		t.Error("CHANGELOG.md written despite the --edit precondition failing")
	}
	log := exec.Command("git", "log", "--oneline")
	log.Dir = dir
	out, _ := log.CombinedOutput()
	if strings.Contains(string(out), "chore(release)") {
		t.Errorf("a release commit was made despite the --edit precondition failing:\n%s", out)
	}
}

func TestFooterTipShownWhileTogglesOff(t *testing.T) {
	tip := footerTip(config.Default("proj"))
	if tip == "" {
		t.Fatal("expected a footer tip while both toggles are off")
	}
	if !strings.Contains(tip, "release.contributors") || !strings.Contains(tip, "release.compare_link") {
		t.Errorf("tip = %q, want it to name both config keys", tip)
	}
}

func TestFooterTipHiddenOnceEnabled(t *testing.T) {
	c := config.Default("proj")
	c.Release.Contributors = true
	if footerTip(c) != "" {
		t.Error("no tip expected once release.contributors is on")
	}

	c = config.Default("proj")
	c.Release.CompareLink = true
	if footerTip(c) != "" {
		t.Error("no tip expected once release.compare_link is on")
	}
}

func TestDoReleaseCutsReleaseCandidate(t *testing.T) {
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
	mk := func(msg string) {
		c := exec.Command("git", "commit", "--allow-empty", "-q", "-m", msg)
		c.Dir = dir
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("commit %q: %v: %s", msg, err, out)
		}
	}
	mk("chore: init")
	tagCmd := exec.Command("git", "tag", "v1.5.0")
	tagCmd.Dir = dir
	if out, err := tagCmd.CombinedOutput(); err != nil {
		t.Fatalf("tag: %v: %s", err, out)
	}
	mk("feat: something big")

	r, err := gitrepo.Open(dir)
	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	f := &releaseFlags{dir: dir, yes: true, rc: true}
	if err := doRelease(&buf, r, config.Default("proj"), f, semver.None, false); err != nil {
		t.Fatalf("doRelease: %v", err)
	}

	if has, _ := r.HasTag("v1.6.0-rc.1"); !has {
		t.Errorf("expected tag v1.6.0-rc.1 to be created\noutput:\n%s", buf.String())
	}
	if !strings.Contains(buf.String(), "this is a pre-release") {
		t.Errorf("missing pre-release notice:\n%s", buf.String())
	}
}
