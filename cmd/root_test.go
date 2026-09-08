package cmd

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"

	"github.com/soyagvs/relio/internal/config"
	"github.com/soyagvs/relio/internal/gitrepo"
	"github.com/soyagvs/relio/internal/semver"
)

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
