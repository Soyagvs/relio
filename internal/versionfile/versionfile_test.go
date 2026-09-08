package versionfile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func write(t *testing.T, dir, name, content string) {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestPlanAndApply(t *testing.T) {
	cases := []struct {
		name    string
		file    string
		content string
		target  Target
		version string
		old     string
		want    string
	}{
		{
			name:    "package.json keeps formatting",
			file:    "package.json",
			content: `{"name":"x","version":"1.5.0","private":true}`,
			target:  Target{Path: "package.json"},
			version: "1.6.0",
			old:     "1.5.0",
			want:    `{"name":"x","version":"1.6.0","private":true}`,
		},
		{
			name:    "nested package.json",
			file:    "web/package.json",
			content: "{\n  \"version\": \"1.5.0\"\n}\n",
			target:  Target{Path: "web/package.json"},
			version: "1.6.0",
			old:     "1.5.0",
			want:    "{\n  \"version\": \"1.6.0\"\n}\n",
		},
		{
			name:    "Cargo.toml first version only",
			file:    "Cargo.toml",
			content: "[package]\nname = \"x\"\nversion = \"1.5.0\"\n\n[dependencies]\nfoo = { version = \"0.1\" }\n",
			target:  Target{Path: "Cargo.toml"},
			version: "1.6.0",
			old:     "1.5.0",
			want:    "[package]\nname = \"x\"\nversion = \"1.6.0\"\n\n[dependencies]\nfoo = { version = \"0.1\" }\n",
		},
		{
			name:    "pyproject.toml PEP 621",
			file:    "pyproject.toml",
			content: "[project]\nname = \"x\"\nversion = \"1.5.0\"\ndependencies = []\n",
			target:  Target{Path: "pyproject.toml"},
			version: "2.0.0",
			old:     "1.5.0",
			want:    "[project]\nname = \"x\"\nversion = \"2.0.0\"\ndependencies = []\n",
		},
		{
			name:    "VERSION file preserves trailing newline",
			file:    "VERSION",
			content: "1.5.0\n",
			target:  Target{Path: "VERSION"},
			version: "1.6.0",
			old:     "1.5.0",
			want:    "1.6.0\n",
		},
		{
			name:    "version.txt with surrounding spaces",
			file:    "version.txt",
			content: "  1.5.0  \n",
			target:  Target{Path: "version.txt"},
			version: "1.6.0",
			old:     "1.5.0",
			want:    "  1.6.0  \n",
		},
		{
			name:    "custom pattern in a python file",
			file:    "pkg/__init__.py",
			content: "name = \"x\"\n__version__ = \"1.5.0\"\n",
			target:  Target{Path: "pkg/__init__.py", Pattern: `__version__ = "([^"]+)"`},
			version: "1.6.0",
			old:     "1.5.0",
			want:    "name = \"x\"\n__version__ = \"1.6.0\"\n",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			write(t, dir, tc.file, tc.content)

			changes, err := Plan(dir, []Target{tc.target}, tc.version)
			if err != nil {
				t.Fatalf("Plan: %v", err)
			}
			if len(changes) != 1 {
				t.Fatalf("got %d changes, want 1", len(changes))
			}
			c := changes[0]
			if c.Old != tc.old || c.New != tc.version {
				t.Errorf("Old/New = %q/%q, want %q/%q", c.Old, c.New, tc.old, tc.version)
			}
			if c.Rel != tc.target.Path {
				t.Errorf("Rel = %q, want %q", c.Rel, tc.target.Path)
			}
			if !filepath.IsAbs(c.Path) {
				t.Errorf("Path = %q, want absolute", c.Path)
			}

			if err := Apply(changes); err != nil {
				t.Fatalf("Apply: %v", err)
			}
			got, err := os.ReadFile(filepath.Join(dir, tc.file))
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tc.want {
				t.Errorf("after Apply =\n%q\nwant\n%q", got, tc.want)
			}
		})
	}
}

// Apply must edit the span the rule matched, not the first occurrence of the
// version string — here the same "1.5.0" appears in a dependency pin above the
// real project version.
func TestApplyEditsMatchedSpanNotFirstOccurrence(t *testing.T) {
	dir := t.TempDir()
	const before = "[build-system]\nrequires = [\"setuptools==1.5.0\"]\n\n[project]\nname = \"x\"\nversion = \"1.5.0\"\n"
	const want = "[build-system]\nrequires = [\"setuptools==1.5.0\"]\n\n[project]\nname = \"x\"\nversion = \"1.6.0\"\n"
	write(t, dir, "pyproject.toml", before)

	changes, err := Plan(dir, []Target{{Path: "pyproject.toml"}}, "1.6.0")
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if err := Apply(changes); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	got, _ := os.ReadFile(filepath.Join(dir, "pyproject.toml"))
	if string(got) != want {
		t.Errorf("after Apply =\n%q\nwant\n%q", got, want)
	}
}

// Apply refuses to write when the file moved under it between Plan and Apply.
func TestApplyDetectsFileDrift(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "VERSION", "1.5.0\n")
	changes, err := Plan(dir, []Target{{Path: "VERSION"}}, "1.6.0")
	if err != nil {
		t.Fatal(err)
	}
	write(t, dir, "VERSION", "9.9.9\n") // someone else edited it

	if err := Apply(changes); err == nil || !strings.Contains(err.Error(), "changed since planning") {
		t.Fatalf("err = %v, want 'changed since planning'", err)
	}
}

func TestPlanDeclaredOrder(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "VERSION", "1.5.0\n")
	write(t, dir, "package.json", `{"version":"1.5.0"}`)

	changes, err := Plan(dir, []Target{{Path: "package.json"}, {Path: "VERSION"}}, "1.6.0")
	if err != nil {
		t.Fatal(err)
	}
	if len(changes) != 2 || changes[0].Rel != "package.json" || changes[1].Rel != "VERSION" {
		t.Fatalf("order not preserved: %+v", changes)
	}
}

func TestPlanMissingFile(t *testing.T) {
	dir := t.TempDir()
	_, err := Plan(dir, []Target{{Path: "package.json"}}, "1.6.0")
	if err == nil || !strings.Contains(err.Error(), "file not found") {
		t.Fatalf("err = %v, want 'file not found'", err)
	}
}

func TestPlanNoMatch(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "package.json", `{"name":"x"}`)
	_, err := Plan(dir, []Target{{Path: "package.json"}}, "1.6.0")
	if err == nil || !strings.Contains(err.Error(), "no version match") {
		t.Fatalf("err = %v, want 'no version match'", err)
	}
}

func TestPlanNoBuiltinRule(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "app.conf", "version = 1.5.0\n")
	_, err := Plan(dir, []Target{{Path: "app.conf"}}, "1.6.0")
	if err == nil || !strings.Contains(err.Error(), "no built-in rule") {
		t.Fatalf("err = %v, want 'no built-in rule'", err)
	}
}

func TestPlanRejectsBadGroupCount(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "a.py", `__version__ = "1.5.0"`)
	for _, p := range []string{`__version__ = "[^"]+"`, `(__version__) = "([^"]+)"`} {
		if _, err := Plan(dir, []Target{{Path: "a.py", Pattern: p}}, "1.6.0"); err == nil {
			t.Errorf("pattern %q: expected an error", p)
		}
	}
}

func TestPlanRejectsUncompilablePattern(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "a.py", "x")
	if _, err := Plan(dir, []Target{{Path: "a.py", Pattern: `([`}}, "1.6.0"); err == nil {
		t.Error("expected a compile error")
	}
}

func TestIdempotentReRun(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "VERSION", "1.6.0\n")

	changes, err := Plan(dir, []Target{{Path: "VERSION"}}, "1.6.0")
	if err != nil {
		t.Fatal(err)
	}
	if len(changes) != 1 || changes[0].Old != changes[0].New {
		t.Fatalf("want Old==New, got %+v", changes)
	}

	before, _ := os.ReadFile(filepath.Join(dir, "VERSION"))
	if err := Apply(changes); err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(filepath.Join(dir, "VERSION"))
	if string(before) != string(after) {
		t.Errorf("bytes changed on idempotent re-run: %q -> %q", before, after)
	}
}

func TestApplyPreservesFilePerm(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "VERSION")
	if err := os.WriteFile(p, []byte("1.5.0\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	changes, err := Plan(dir, []Target{{Path: "VERSION"}}, "1.6.0")
	if err != nil {
		t.Fatal(err)
	}
	if err := Apply(changes); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("perm = %o, want 600", info.Mode().Perm())
	}
}
