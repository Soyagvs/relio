package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSaveAndLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	want := Default("azeink")

	if err := want.Save(dir); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if !Exists(dir) {
		t.Fatal("Exists = false after Save")
	}

	got, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Project != "azeink" {
		t.Errorf("Project = %q", got.Project)
	}
	if !got.Release.ChangelogEnabled() || !got.Release.TagEnabled() {
		t.Errorf("release flags lost: %+v", got.Release)
	}
	if got.Release.TagPrefix != "v" || got.Release.ChangelogFile != "CHANGELOG.md" {
		t.Errorf("release defaults lost: %+v", got.Release)
	}
}

func TestSaveRefusesOverwrite(t *testing.T) {
	dir := t.TempDir()
	if err := Default("x").Save(dir); err != nil {
		t.Fatalf("first Save: %v", err)
	}
	if err := Default("x").Save(dir); err == nil {
		t.Error("expected error on second Save")
	}
}

func TestLoadMissing(t *testing.T) {
	if _, err := Load(t.TempDir()); err != ErrNotFound {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestLoadFillsDefaults(t *testing.T) {
	dir := t.TempDir()
	minimal := "project: tiny\n"
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte(minimal), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Versioning != "semver" || got.Commits != "conventional" {
		t.Errorf("defaults not applied: %+v", got)
	}
	if got.Release.ChangelogFile != "CHANGELOG.md" || got.Release.TagPrefix != "v" {
		t.Errorf("release defaults not applied: %+v", got.Release)
	}
	// A file with no changelog/tag keys still releases fully.
	if !got.Release.ChangelogEnabled() || !got.Release.TagEnabled() {
		t.Errorf("minimal config should enable changelog and tag: %+v", got.Release)
	}
}

func TestExplicitReleaseFalseIsHonoured(t *testing.T) {
	dir := t.TempDir()
	body := "project: x\nrelease:\n    tag: false\n"
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Release.TagEnabled() {
		t.Error("explicit `tag: false` was ignored")
	}
	if !got.Release.ChangelogEnabled() {
		t.Error("`changelog:` was omitted and should default to true")
	}
}

func TestGitHubConfigRoundTrip(t *testing.T) {
	dir := t.TempDir()
	want := Default("x")
	want.GitHub.Enabled = true
	want.GitHub.Repo = "acme/relio"
	want.GitHub.Release = true

	if err := want.Save(dir); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.GitHub != want.GitHub {
		t.Errorf("GitHub = %+v, want %+v", got.GitHub, want.GitHub)
	}
}

func TestLoadWithoutGitHubFields(t *testing.T) {
	dir := t.TempDir()
	old := "project: legacy\nrelease:\n  changelog: true\n  tag: true\n"
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte(old), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.GitHub.Enabled || got.GitHub.Repo != "" || got.GitHub.Release {
		t.Errorf("GitHub should default to zero values, got %+v", got.GitHub)
	}
}

func TestVersionFilesMixedListLoads(t *testing.T) {
	dir := t.TempDir()
	yml := "project: x\nrelease:\n" +
		"  version_files:\n" +
		"    - package.json\n" +
		"    - {path: pyproject.toml, pattern: 'version = \"([^\"]+)\"'}\n"
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte(yml), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	vf := got.Release.VersionFiles
	if len(vf) != 2 {
		t.Fatalf("VersionFiles = %+v, want 2", vf)
	}
	if vf[0].Path != "package.json" || vf[0].Pattern != "" {
		t.Errorf("vf[0] = %+v, want {package.json, \"\"}", vf[0])
	}
	if vf[1].Path != "pyproject.toml" || vf[1].Pattern != `version = "([^"]+)"` {
		t.Errorf("vf[1] = %+v", vf[1])
	}

	targets := got.Release.VersionTargets()
	if len(targets) != 2 || targets[0].Path != "package.json" || targets[1].Pattern != `version = "([^"]+)"` {
		t.Errorf("VersionTargets = %+v", targets)
	}
}

func TestVersionFilesRoundTrip(t *testing.T) {
	dir := t.TempDir()
	want := Default("x")
	want.Release.VersionFiles = []VersionFile{
		{Path: "package.json"},
		{Path: "src/app.py", Pattern: `__version__ = "([^"]+)"`},
	}
	if err := want.Save(dir); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(got.Release.VersionFiles) != 2 {
		t.Fatalf("VersionFiles = %+v", got.Release.VersionFiles)
	}
	if got.Release.VersionFiles[0] != (VersionFile{Path: "package.json"}) {
		t.Errorf("vf[0] = %+v", got.Release.VersionFiles[0])
	}
	if got.Release.VersionFiles[1] != (VersionFile{Path: "src/app.py", Pattern: `__version__ = "([^"]+)"`}) {
		t.Errorf("vf[1] = %+v", got.Release.VersionFiles[1])
	}
}

func TestLoadWithoutVersionFiles(t *testing.T) {
	dir := t.TempDir()
	old := "project: legacy\nrelease:\n  changelog: true\n  tag: true\n"
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte(old), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Release.VersionFiles != nil {
		t.Errorf("VersionFiles = %+v, want nil", got.Release.VersionFiles)
	}
	if got.Release.VersionTargets() != nil {
		t.Errorf("VersionTargets = %+v, want nil", got.Release.VersionTargets())
	}
}

func TestHooksScalarAndList(t *testing.T) {
	dir := t.TempDir()
	yml := "project: x\nrelease:\n" +
		"  hooks:\n" +
		"    before: make test\n" +
		"    after:\n" +
		"      - ./scripts/notify.sh\n" +
		"      - echo done\n"
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte(yml), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(got.Release.Hooks.Before) != 1 || got.Release.Hooks.Before[0] != "make test" {
		t.Errorf("Before = %+v, want [make test]", got.Release.Hooks.Before)
	}
	want := []string{"./scripts/notify.sh", "echo done"}
	if len(got.Release.Hooks.After) != 2 || got.Release.Hooks.After[0] != want[0] || got.Release.Hooks.After[1] != want[1] {
		t.Errorf("After = %+v, want %+v", got.Release.Hooks.After, want)
	}
}

func TestHooksRoundTrip(t *testing.T) {
	dir := t.TempDir()
	want := Default("x")
	want.Release.Hooks = HooksConfig{
		Before: StringList{"make test"},
		After:  StringList{"./scripts/notify.sh", "echo done"},
	}
	if err := want.Save(dir); err != nil {
		t.Fatalf("Save: %v", err)
	}
	data, err := os.ReadFile(Path(dir))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "before: make test") {
		t.Errorf("marshaled YAML should carry `before: make test` as a scalar:\n%s", data)
	}
	got, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(got.Release.Hooks.Before) != 1 || got.Release.Hooks.Before[0] != "make test" {
		t.Errorf("Before = %+v", got.Release.Hooks.Before)
	}
	if len(got.Release.Hooks.After) != 2 {
		t.Errorf("After = %+v, want 2 entries", got.Release.Hooks.After)
	}
}

func TestLoadWithoutHooks(t *testing.T) {
	dir := t.TempDir()
	old := "project: legacy\nrelease:\n  changelog: true\n  tag: true\n"
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte(old), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Release.Hooks.Before != nil || got.Release.Hooks.After != nil {
		t.Errorf("Hooks = %+v, want nil slices", got.Release.Hooks)
	}
}

func TestLoadRejectsUnsupportedVersioning(t *testing.T) {
	dir := t.TempDir()
	bad := "project: x\nversioning: calver\n"
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte(bad), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(dir); err == nil {
		t.Error("expected error for calver")
	}
}
