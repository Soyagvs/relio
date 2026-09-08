package hook

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestRunSuccess(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("assertions use the `sh` shell")
	}
	dir := t.TempDir()
	var buf bytes.Buffer
	if err := Run(&buf, dir, []string{"echo hi > out.txt"}, Env{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(dir, "out.txt"))
	if err != nil {
		t.Fatalf("reading out.txt: %v", err)
	}
	if string(got) != "hi\n" {
		t.Errorf("out.txt = %q, want %q", got, "hi\n")
	}
}

func TestRunStopsOnFirstFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("assertions use the `sh` shell")
	}
	dir := t.TempDir()
	var buf bytes.Buffer
	err := Run(&buf, dir, []string{"echo one > log", "exit 3", "echo two >> log"}, Env{})
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if !strings.Contains(err.Error(), "exit") || !strings.Contains(err.Error(), "3") {
		t.Errorf("error = %q, want it to mention exit status 3", err)
	}
	log, rerr := os.ReadFile(filepath.Join(dir, "log"))
	if rerr != nil {
		t.Fatalf("reading log: %v", rerr)
	}
	if !strings.Contains(string(log), "one") {
		t.Errorf("log = %q, want it to contain %q", log, "one")
	}
	if strings.Contains(string(log), "two") {
		t.Errorf("log = %q, third command ran after a failure", log)
	}
}

func TestRunExportsEnv(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("assertions use the `sh` shell")
	}
	dir := t.TempDir()
	var buf bytes.Buffer
	if err := Run(&buf, dir, []string{`printf %s "$RELIO_TAG" > tag.txt`}, Env{Tag: "v9.9.9"}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(dir, "tag.txt"))
	if err != nil {
		t.Fatalf("reading tag.txt: %v", err)
	}
	if string(got) != "v9.9.9" {
		t.Errorf("tag.txt = %q, want %q", got, "v9.9.9")
	}
}

func TestRunUsesWorkingDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("assertions use the `sh` shell")
	}
	dir := t.TempDir()
	var buf bytes.Buffer
	if err := Run(&buf, dir, []string{"echo here > relative.txt"}, Env{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "relative.txt")); err != nil {
		t.Errorf("relative file not created under dir: %v", err)
	}
}

func TestRunEmptyListIsNoOp(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("assertions use the `sh` shell")
	}
	var buf bytes.Buffer
	if err := Run(&buf, t.TempDir(), nil, Env{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("wrote %q, want no output for an empty list", buf.String())
	}
}
