package userconfig

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestLoadMissingFileReturnsZeroConfig(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if got := Load(); got != (Config{}) {
		t.Errorf("Load() = %+v, want zero Config", got)
	}
}

func TestLoadGarbageFileReturnsZeroConfig(t *testing.T) {
	base := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", base)
	dir := filepath.Join(base, "relio")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("not: [valid: yaml"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := Load(); got != (Config{}) {
		t.Errorf("Load() with a garbage file = %+v, want zero Config", got)
	}
}

func TestSaveThenLoadRoundTrip(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	if err := Save(Config{Language: "es"}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if got := Load(); got.Language != "es" {
		t.Errorf("Load().Language = %q, want es", got.Language)
	}
}

func TestSaveCreatesMissingDirectory(t *testing.T) {
	base := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", base)

	dir, err := Dir()
	if err != nil {
		t.Fatal(err)
	}
	if _, statErr := os.Stat(dir); !os.IsNotExist(statErr) {
		t.Fatalf("precondition: %s already exists", dir)
	}

	if err := Save(Config{Language: "en"}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if _, statErr := os.Stat(dir); statErr != nil {
		t.Errorf("Save did not create %s: %v", dir, statErr)
	}
}

func TestSaveOverwritesOnSecondCall(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	if err := Save(Config{Language: "en"}); err != nil {
		t.Fatalf("first Save: %v", err)
	}
	if err := Save(Config{Language: "es"}); err != nil {
		t.Fatalf("second Save: %v", err)
	}
	if got := Load().Language; got != "es" {
		t.Errorf("Load().Language after overwrite = %q, want es", got)
	}
}

func TestSaveUnwritableDirReturnsErrorNotPanic(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("chmod semantics differ on windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("running as root bypasses permission checks")
	}
	base := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", base)

	if err := os.Chmod(base, 0o500); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(base, 0o700)

	if err := Save(Config{Language: "es"}); err == nil {
		t.Fatal("expected an error saving under an unwritable parent directory, got nil")
	}
}
