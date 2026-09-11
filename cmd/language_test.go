package cmd

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/soyagvs/relio/internal/i18n"
	"github.com/soyagvs/relio/internal/userconfig"
)

// initRepoWithConfig creates a bare temp git repo and, when releaseYAML is
// non-empty, writes it as .release.yaml at the repo root.
func initRepoWithConfig(t *testing.T, releaseYAML string) string {
	t.Helper()
	dir := t.TempDir()
	for _, args := range [][]string{
		{"init", "-q"},
		{"config", "user.email", "t@e.com"},
		{"config", "user.name", "T"},
	} {
		c := exec.Command("git", args...)
		c.Dir = dir
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	if releaseYAML != "" {
		if err := os.WriteFile(filepath.Join(dir, ".release.yaml"), []byte(releaseYAML), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestResolveLanguageDefaultsToEnglish(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	defer i18n.SetLanguage("en")

	dir := initRepoWithConfig(t, "")
	var buf bytes.Buffer
	if got := resolveLanguage(dir, &buf); got != "en" {
		t.Errorf("resolveLanguage = %q, want en", got)
	}
	if i18n.Current() != "en" {
		t.Errorf("i18n.Current() = %q, want en", i18n.Current())
	}
	if buf.String() != "" {
		t.Errorf("unexpected warning output: %q", buf.String())
	}
}

func TestResolveLanguageGlobalConfigOverridesDefault(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	defer i18n.SetLanguage("en")
	if err := userconfig.Save(userconfig.Config{Language: "es"}); err != nil {
		t.Fatalf("userconfig.Save: %v", err)
	}

	dir := initRepoWithConfig(t, "")
	var buf bytes.Buffer
	if got := resolveLanguage(dir, &buf); got != "es" {
		t.Errorf("resolveLanguage = %q, want es", got)
	}
}

func TestResolveLanguageReleaseYamlOverridesGlobalConfig(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	defer i18n.SetLanguage("en")
	if err := userconfig.Save(userconfig.Config{Language: "es"}); err != nil {
		t.Fatalf("userconfig.Save: %v", err)
	}

	dir := initRepoWithConfig(t, "project: x\nlanguage: en\n")
	var buf bytes.Buffer
	if got := resolveLanguage(dir, &buf); got != "en" {
		t.Errorf("resolveLanguage = %q, want en (.release.yaml override wins over global)", got)
	}
}

func TestResolveLanguageOutsideRepoStillAppliesGlobalConfig(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	defer i18n.SetLanguage("en")
	if err := userconfig.Save(userconfig.Config{Language: "es"}); err != nil {
		t.Fatalf("userconfig.Save: %v", err)
	}

	var buf bytes.Buffer
	if got := resolveLanguage(t.TempDir(), &buf); got != "es" {
		t.Errorf("resolveLanguage(non-repo dir) = %q, want es (project override is skipped, global still applies)", got)
	}
}

func TestResolveLanguageUnknownIDWarnsOnceAndFallsBack(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	defer i18n.SetLanguage("en")
	if err := userconfig.Save(userconfig.Config{Language: "fr"}); err != nil {
		t.Fatalf("userconfig.Save: %v", err)
	}

	dir := initRepoWithConfig(t, "")
	var buf bytes.Buffer
	if got := resolveLanguage(dir, &buf); got != "en" {
		t.Errorf("resolveLanguage = %q, want en (unknown id falls back)", got)
	}
	out := buf.String()
	if n := strings.Count(out, "\n"); n != 1 {
		t.Errorf("expected exactly one warning line, got %d lines:\n%s", n, out)
	}
	if !strings.Contains(out, "fr") {
		t.Errorf("warning should name the unknown language id %q:\n%s", "fr", out)
	}
}

func TestPrescanDirFlagLongForm(t *testing.T) {
	if got := prescanDir([]string{"--dir", "/tmp/x"}); got != "/tmp/x" {
		t.Errorf("prescanDir(--dir x) = %q, want /tmp/x", got)
	}
}

func TestPrescanDirFlagShortForm(t *testing.T) {
	if got := prescanDir([]string{"-C", "/tmp/y"}); got != "/tmp/y" {
		t.Errorf("prescanDir(-C y) = %q, want /tmp/y", got)
	}
}

func TestPrescanDirFlagEqualsForm(t *testing.T) {
	if got := prescanDir([]string{"status", "--dir=/tmp/z"}); got != "/tmp/z" {
		t.Errorf("prescanDir(status --dir=/tmp/z) = %q, want /tmp/z", got)
	}
}

func TestPrescanDirDefaultsToCurrentDir(t *testing.T) {
	if got := prescanDir([]string{"status", "--yes"}); got != "." {
		t.Errorf("prescanDir(no dir flag) = %q, want .", got)
	}
}

func TestPrescanDirIgnoresTrailingFlagWithNoValue(t *testing.T) {
	if got := prescanDir([]string{"status", "--dir"}); got != "." {
		t.Errorf("prescanDir(dangling --dir) = %q, want . (no value to prescan)", got)
	}
}
