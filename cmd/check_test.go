package cmd

import (
	"bytes"
	"io"
	"os/exec"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/soyagvs/relio/internal/config"
	"github.com/soyagvs/relio/internal/conventional"
	"github.com/soyagvs/relio/internal/gitrepo"
	"github.com/soyagvs/relio/internal/release"
	"github.com/soyagvs/relio/internal/ui"
)

func checkCommits() (conv, nonConv []conventional.Commit) {
	all := []conventional.Commit{
		{Type: "feat", Description: "add login screen", Raw: "feat: add login screen", Hash: "a1b2c3ddddddddddddddddddddddddddddddddddd"},
		{Type: "fix", Description: "crash", Raw: "fix: crash", Hash: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"},
		{Raw: "dashboard fix", Description: "dashboard fix", Hash: "d4e5f6adddddddddddddddddddddddddddddddddd"},
		{Raw: "wip", Description: "wip", Hash: "f7g8h9idddddddddddddddddddddddddddddddddd"},
	}
	for _, c := range all {
		if c.IsConventional() {
			conv = append(conv, c)
		} else {
			nonConv = append(nonConv, c)
		}
	}
	return conv, nonConv
}

func TestCheckReportCountsAndBlocks(t *testing.T) {
	conv, nonConv := checkCommits()
	var buf bytes.Buffer
	if err := checkReport(&buf, "relio", "v1.5.0", 4, conv, nonConv, "minor", "v1.6.0", false); err != nil {
		t.Fatal(err)
	}
	got := buf.String()

	for _, want := range []string{
		"relio",
		"4 commits since v1.5.0",
		"2 conventional",
		"2 not conventional:",
		"dashboard fix",
		"wip",
		"Detected bump: minor  →  v1.6.0",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
	// short hash column, 7 chars
	if !strings.Contains(got, "d4e5f6a  dashboard fix") {
		t.Errorf("expected 7-char hash column:\n%s", got)
	}
	if strings.Contains(got, "d4e5f6adddddddddddddddddddddddddddddddddd") {
		t.Errorf("full hash leaked:\n%s", got)
	}
}

func TestCheckReportCleanHistoryOmitsCrossBlock(t *testing.T) {
	conv, _ := checkCommits()
	var buf bytes.Buffer
	if err := checkReport(&buf, "relio", "v1.5.0", len(conv), conv, nil, "minor", "v1.6.0", false); err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	if !strings.Contains(got, "2 conventional") {
		t.Errorf("still want the check line:\n%s", got)
	}
	if strings.Contains(got, "not conventional") {
		t.Errorf("cross block must be omitted when everything is conventional:\n%s", got)
	}
}

func TestCheckReportNoCommits(t *testing.T) {
	var buf bytes.Buffer
	if err := checkReport(&buf, "relio", "v1.5.0", 0, nil, nil, "none", "v1.5.0", false); err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	if !strings.Contains(got, "Nothing to check — no commits since v1.5.0.") {
		t.Errorf("want the nothing-to-check line:\n%s", got)
	}
	if strings.Contains(got, "Detected bump") {
		t.Errorf("no bump line when there is nothing to check:\n%s", got)
	}
}

func TestCheckReportNoTagYet(t *testing.T) {
	conv, nonConv := checkCommits()
	var buf bytes.Buffer
	if err := checkReport(&buf, "relio", "", 4, conv, nonConv, "minor", "v0.1.0", false); err != nil {
		t.Fatal(err)
	}
	if got := buf.String(); !strings.Contains(got, "4 commits (no tag yet)") {
		t.Errorf("want the no-tag header:\n%s", got)
	}
}

func TestCheckReportNoHashDropsColumn(t *testing.T) {
	conv, nonConv := checkCommits()
	var buf bytes.Buffer
	if err := checkReport(&buf, "relio", "v1.5.0", 4, conv, nonConv, "minor", "v1.6.0", true); err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	if strings.Contains(got, "d4e5f6a") || strings.Contains(got, "f7g8h9i") {
		t.Errorf("--no-hash must drop the hash column:\n%s", got)
	}
	if !strings.Contains(got, "dashboard fix") || !strings.Contains(got, "wip") {
		t.Errorf("subjects must still be listed:\n%s", got)
	}
}

// --- runCheck against a real temp repo ---

func newCheckRepo(t *testing.T) (string, *gitrepo.Repo) {
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

func checkCommit(t *testing.T, dir, msg string) {
	t.Helper()
	c := exec.Command("git", "commit", "--allow-empty", "-q", "-m", msg)
	c.Dir = dir
	if out, err := c.CombinedOutput(); err != nil {
		t.Fatalf("commit %q: %v: %s", msg, err, out)
	}
}

func runCheckPlan(t *testing.T, r *gitrepo.Repo, cfg config.Config, strict bool) error {
	t.Helper()
	plan, err := release.BuildPlan(r, cfg, release.Options{})
	if err != nil {
		t.Fatal(err)
	}
	cmd := &cobra.Command{}
	cmd.SetOut(io.Discard)
	return runCheck(cmd, r, cfg, plan, strict)
}

func TestRunCheckStrictFailsOnNonConventional(t *testing.T) {
	dir, r := newCheckRepo(t)
	checkCommit(t, dir, "feat: proper one")
	checkCommit(t, dir, "just a wip commit")
	cfg := config.Default("proj")

	err := runCheckPlan(t, r, cfg, true)
	if err == nil {
		t.Fatal("strict: want a non-nil error with a non-conventional commit")
	}
	if !strings.Contains(err.Error(), "not Conventional Commits (--strict)") {
		t.Errorf("err = %v", err)
	}

	if err := runCheckPlan(t, r, cfg, false); err != nil {
		t.Errorf("non-strict: want nil, got %v", err)
	}
}

func TestRunCheckStrictPassesWhenAllConventional(t *testing.T) {
	dir, r := newCheckRepo(t)
	checkCommit(t, dir, "feat: one")
	checkCommit(t, dir, "fix: two")
	cfg := config.Default("proj")

	if err := runCheckPlan(t, r, cfg, true); err != nil {
		t.Errorf("strict + all conventional: want nil, got %v", err)
	}
}

func TestRunCheckNoCommitsIsQuietEvenWithStrict(t *testing.T) {
	dir, r := newCheckRepo(t)
	checkCommit(t, dir, "chore: init")
	if err := r.CreateTag("v1.0.0", "release v1.0.0"); err != nil {
		t.Fatal(err)
	}
	cfg := config.Default("proj")

	if err := runCheckPlan(t, r, cfg, true); err != nil {
		t.Errorf("no commits since the tag: want nil even with --strict, got %v", err)
	}
}

func TestNewCheckCmdShape(t *testing.T) {
	c := newCheckCmd(&releaseFlags{})
	if c.Use != "check" {
		t.Errorf("Use = %q", c.Use)
	}
	if c.Flags().Lookup("strict") == nil {
		t.Error("missing --strict flag")
	}
	_ = ui.HideHashes
}
