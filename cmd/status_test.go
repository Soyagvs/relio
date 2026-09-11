package cmd

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/soyagvs/relio/internal/config"
	"github.com/soyagvs/relio/internal/gitrepo"
	"github.com/soyagvs/relio/internal/i18n"
)

func newStatusRepo(t *testing.T) (string, *gitrepo.Repo) {
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

func statusCommit(t *testing.T, dir, msg string) {
	t.Helper()
	c := exec.Command("git", "commit", "--allow-empty", "-q", "-m", msg)
	c.Dir = dir
	if out, err := c.CombinedOutput(); err != nil {
		t.Fatalf("commit %q: %v: %s", msg, err, out)
	}
}

func statusTag(t *testing.T, dir, name string) {
	t.Helper()
	c := exec.Command("git", "tag", name)
	c.Dir = dir
	if out, err := c.CombinedOutput(); err != nil {
		t.Fatalf("tag %q: %v: %s", name, err, out)
	}
}

func TestRunStatusPrereleaseHint(t *testing.T) {
	dir, r := newStatusRepo(t)
	statusCommit(t, dir, "chore: init")
	statusTag(t, dir, "v1.5.0")
	statusCommit(t, dir, "feat: something")
	statusTag(t, dir, "v1.6.0-rc.1")
	statusCommit(t, dir, "fix: a bug")

	var buf bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&buf)
	if err := runStatus(cmd, r, config.Default("proj")); err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	if !strings.Contains(got, "on a pre-release") || !strings.Contains(got, "`relio` finalizes v1.6.0") {
		t.Errorf("missing pre-release hint:\n%s", got)
	}
}

func TestRunStatusNoPrereleaseHintOnStable(t *testing.T) {
	dir, r := newStatusRepo(t)
	statusCommit(t, dir, "chore: init")
	statusTag(t, dir, "v1.5.0")
	statusCommit(t, dir, "feat: something")

	var buf bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&buf)
	if err := runStatus(cmd, r, config.Default("proj")); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(buf.String(), "on a pre-release") {
		t.Errorf("unexpected pre-release hint on a stable current:\n%s", buf.String())
	}
}

// --- i18n: construction-time Short + runtime output localization ---

func TestNewStatusCmdShortLocalizesAtConstructionTime(t *testing.T) {
	prev := i18n.Current()
	t.Cleanup(func() { i18n.SetLanguage(prev) })

	i18n.SetLanguage("en")
	shortEN := newStatusCmd(&releaseFlags{}).Short

	i18n.SetLanguage("es")
	shortES := newStatusCmd(&releaseFlags{}).Short

	if shortEN != "Show what's unreleased and the version it suggests" {
		t.Errorf("newStatusCmd().Short (en) = %q", shortEN)
	}
	if shortES == shortEN || shortES == "" {
		t.Errorf("newStatusCmd().Short unchanged across languages: %q", shortES)
	}
}

func TestRunStatusLocalizesOutput(t *testing.T) {
	dir, r := newStatusRepo(t)
	statusCommit(t, dir, "feat: something")

	prev := i18n.Current()
	t.Cleanup(func() { i18n.SetLanguage(prev) })

	i18n.SetLanguage("en")
	var bufEN bytes.Buffer
	cmdEN := &cobra.Command{}
	cmdEN.SetOut(&bufEN)
	if err := runStatus(cmdEN, r, config.Default("proj")); err != nil {
		t.Fatal(err)
	}

	i18n.SetLanguage("es")
	var bufES bytes.Buffer
	cmdES := &cobra.Command{}
	cmdES.SetOut(&bufES)
	if err := runStatus(cmdES, r, config.Default("proj")); err != nil {
		t.Fatal(err)
	}

	if bufEN.String() == bufES.String() {
		t.Error("runStatus output unchanged across languages")
	}
	for _, want := range []string{"Current", "Unreleased", "Suggested", "Ready to release."} {
		if !strings.Contains(bufEN.String(), want) {
			t.Errorf("english golden text %q missing:\n%s", want, bufEN.String())
		}
	}
}

func TestRunStatusNothingToReleaseLocalizesOutput(t *testing.T) {
	dir, r := newStatusRepo(t)
	statusCommit(t, dir, "chore: init")
	statusTag(t, dir, "v1.0.0")

	prev := i18n.Current()
	t.Cleanup(func() { i18n.SetLanguage(prev) })

	i18n.SetLanguage("en")
	var bufEN bytes.Buffer
	cmdEN := &cobra.Command{}
	cmdEN.SetOut(&bufEN)
	if err := runStatus(cmdEN, r, config.Default("proj")); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(bufEN.String(), "Nothing to release.") {
		t.Errorf("english golden text missing:\n%s", bufEN.String())
	}

	i18n.SetLanguage("es")
	var bufES bytes.Buffer
	cmdES := &cobra.Command{}
	cmdES.SetOut(&bufES)
	if err := runStatus(cmdES, r, config.Default("proj")); err != nil {
		t.Fatal(err)
	}

	if bufEN.String() == bufES.String() {
		t.Error("runStatus nothing-to-release output unchanged across languages")
	}
}
