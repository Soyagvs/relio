package config

import (
	"os"
	"path/filepath"
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
	if !got.Release.Changelog || !got.Release.Tag {
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
