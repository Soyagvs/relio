package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// copyFixture copies a testdata fixture into dir/.release.yaml and returns its
// path. mode, when non-zero, is applied after the copy so tests can assert
// SetFields preserves the original file mode.
func copyFixture(t *testing.T, dir, fixture string, mode os.FileMode) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", fixture))
	if err != nil {
		t.Fatalf("reading fixture %s: %v", fixture, err)
	}
	p := filepath.Join(dir, FileName)
	if mode == 0 {
		mode = 0o644
	}
	if err := os.WriteFile(p, data, mode); err != nil {
		t.Fatalf("writing fixture %s: %v", fixture, err)
	}
	return p
}

// TestSetFieldsDoesNotMaterializeOmittedBoolPointers is the load-bearing test
// for this slice: Release.Changelog / Release.Tag are *bool with
// nil-means-default semantics (see ChangelogEnabled/TagEnabled). A file that
// omits both keys MUST still omit them after SetFields touches an unrelated
// sibling key — a full yaml.Marshal(Config) rewrite would instead emit
// `changelog: null` / `tag: null`, which is a correctness bug, not a
// formatting nit.
func TestSetFieldsDoesNotMaterializeOmittedBoolPointers(t *testing.T) {
	dir := t.TempDir()
	copyFixture(t, dir, "edit_comments.yaml", 0)

	if err := SetFields(dir,
		Field{Path: []string{"release", "contributors"}, Value: true},
		Field{Path: []string{"release", "compare_link"}, Value: true},
	); err != nil {
		t.Fatalf("SetFields: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(dir, FileName))
	if err != nil {
		t.Fatal(err)
	}
	out := string(got)

	if strings.Contains(out, "changelog") {
		t.Errorf("changelog key materialized, output:\n%s", out)
	}
	if strings.Contains(out, "tag") {
		t.Errorf("tag key materialized, output:\n%s", out)
	}
	if !strings.Contains(out, "contributors: true") {
		t.Errorf("contributors: true not found, output:\n%s", out)
	}
	if !strings.Contains(out, "compare_link: true") {
		t.Errorf("compare_link: true not found, output:\n%s", out)
	}

	// Round-trip through Config to prove the *bool semantics actually hold,
	// not just that the substring "changelog" is absent.
	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load after SetFields: %v", err)
	}
	if cfg.Release.Changelog != nil {
		t.Errorf("Release.Changelog = %v, want nil (default-on)", *cfg.Release.Changelog)
	}
	if cfg.Release.Tag != nil {
		t.Errorf("Release.Tag = %v, want nil (default-on)", *cfg.Release.Tag)
	}
	if !cfg.Release.ChangelogEnabled() || !cfg.Release.TagEnabled() {
		t.Errorf("release flags no longer default-on: %+v", cfg.Release)
	}
}

// TestSetFieldsDoesNotMaterializeAbsentSiblingBlocks proves that top-level
// blocks absent from the source file (github:, content:) are not
// materialized as empty mappings by touching an unrelated top-level key.
func TestSetFieldsDoesNotMaterializeAbsentSiblingBlocks(t *testing.T) {
	dir := t.TempDir()
	copyFixture(t, dir, "edit_comments.yaml", 0)

	if err := SetFields(dir, Field{Path: []string{"language"}, Value: "es"}); err != nil {
		t.Fatalf("SetFields: %v", err)
	}

	out, err := os.ReadFile(filepath.Join(dir, FileName))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(out), "github") {
		t.Errorf("github block materialized, output:\n%s", out)
	}
	if strings.Contains(string(out), "content") {
		t.Errorf("content block materialized, output:\n%s", out)
	}
	if !strings.Contains(string(out), "language: es") {
		t.Errorf("language: es not found, output:\n%s", out)
	}
}

// TestSetFieldsCreatesMissingKey covers creating a brand-new top-level key.
func TestSetFieldsCreatesMissingKey(t *testing.T) {
	dir := t.TempDir()
	copyFixture(t, dir, "edit_comments.yaml", 0)

	if err := SetFields(dir, Field{Path: []string{"language"}, Value: "es"}); err != nil {
		t.Fatalf("SetFields: %v", err)
	}

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Language != "es" {
		t.Errorf("Language = %q, want es", cfg.Language)
	}
}

// TestSetFieldsUpdatesExistingValueInPlace covers overwriting an existing
// scalar value without disturbing its sibling.
func TestSetFieldsUpdatesExistingValueInPlace(t *testing.T) {
	dir := t.TempDir()
	copyFixture(t, dir, "edit_comments.yaml", 0)

	if err := SetFields(dir, Field{Path: []string{"release", "compare_link"}, Value: true}); err != nil {
		t.Fatalf("SetFields: %v", err)
	}

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !cfg.Release.CompareLink {
		t.Error("CompareLink not updated to true")
	}
	if cfg.Release.Contributors {
		t.Error("Contributors was disturbed by an unrelated SetFields call")
	}
}

// TestSetFieldsPreservesComments proves HeadComment/LineComment on the target
// key AND on sibling keys survive an in-place update.
func TestSetFieldsPreservesComments(t *testing.T) {
	dir := t.TempDir()
	copyFixture(t, dir, "edit_comments.yaml", 0)

	if err := SetFields(dir, Field{Path: []string{"release", "compare_link"}, Value: true}); err != nil {
		t.Fatalf("SetFields: %v", err)
	}

	out, err := os.ReadFile(filepath.Join(dir, FileName))
	if err != nil {
		t.Fatal(err)
	}
	got := string(out)

	if !strings.Contains(got, "# release configuration — safe to commit. Never put secrets here.") {
		t.Errorf("top-of-file comment lost, output:\n%s", got)
	}
	if !strings.Contains(got, "# keep the compare link off for now") {
		t.Errorf("line comment on the updated key lost, output:\n%s", got)
	}
	if !strings.Contains(got, "# contributors line toggle") {
		t.Errorf("head comment on the sibling key lost, output:\n%s", got)
	}
}

// TestSetFieldsPreservesKeyOrder proves unrelated keys keep their relative
// order after a write.
func TestSetFieldsPreservesKeyOrder(t *testing.T) {
	dir := t.TempDir()
	copyFixture(t, dir, "edit_comments.yaml", 0)

	if err := SetFields(dir, Field{Path: []string{"release", "compare_link"}, Value: true}); err != nil {
		t.Fatalf("SetFields: %v", err)
	}

	out, err := os.ReadFile(filepath.Join(dir, FileName))
	if err != nil {
		t.Fatal(err)
	}
	got := string(out)

	iProject := strings.Index(got, "project:")
	iVersioning := strings.Index(got, "versioning:")
	iCommits := strings.Index(got, "commits:")
	iRelease := strings.Index(got, "release:")
	if iProject < 0 || iVersioning < 0 || iCommits < 0 || iRelease < 0 {
		t.Fatalf("expected keys missing from output:\n%s", got)
	}
	if !(iProject < iVersioning && iVersioning < iCommits && iCommits < iRelease) {
		t.Errorf("key order disturbed, output:\n%s", got)
	}
}

// TestSetFieldsCreatesMissingReleaseBlock covers the intermediate-mapping
// creation path when release: is entirely absent from the source file.
func TestSetFieldsCreatesMissingReleaseBlock(t *testing.T) {
	dir := t.TempDir()
	copyFixture(t, dir, "edit_missing_release.yaml", 0)

	if err := SetFields(dir, Field{Path: []string{"release", "contributors"}, Value: true}); err != nil {
		t.Fatalf("SetFields: %v", err)
	}

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !cfg.Release.Contributors {
		t.Error("Contributors not set after creating the release block")
	}
	if cfg.Release.Changelog != nil || cfg.Release.Tag != nil {
		t.Errorf("release block creation should not populate changelog/tag: %+v", cfg.Release)
	}
}

// TestSetFieldsEmptyExistingFile covers a zero-byte .release.yaml that
// already exists (distinct from a missing file, see
// TestSetFieldsMissingFileReturnsErrNotFound).
func TestSetFieldsEmptyExistingFile(t *testing.T) {
	dir := t.TempDir()
	copyFixture(t, dir, "edit_empty.yaml", 0)

	if err := SetFields(dir, Field{Path: []string{"language"}, Value: "es"}); err != nil {
		t.Fatalf("SetFields: %v", err)
	}

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Language != "es" {
		t.Errorf("Language = %q, want es", cfg.Language)
	}
}

// TestSetFieldsMissingFileReturnsErrNotFound documents the chosen behavior
// for a file that does not exist at all: SetFields errors with ErrNotFound,
// consistent with Load's existing behavior for the same case. SetFields
// modifies an existing config; callers that want create-on-first-write
// call Save first (mirroring Save's own "refuses to overwrite" contract on
// the other side of the same lifecycle).
func TestSetFieldsMissingFileReturnsErrNotFound(t *testing.T) {
	dir := t.TempDir()

	if err := SetFields(dir, Field{Path: []string{"language"}, Value: "es"}); err != ErrNotFound {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

// TestSetFieldsNonMappingRootErrors covers a .release.yaml whose document
// root is not a mapping (e.g. a bare sequence). SetFields must refuse rather
// than silently overwrite the file.
func TestSetFieldsNonMappingRootErrors(t *testing.T) {
	dir := t.TempDir()
	copyFixture(t, dir, "edit_non_mapping.yaml", 0)
	before, err := os.ReadFile(filepath.Join(dir, FileName))
	if err != nil {
		t.Fatal(err)
	}

	if err := SetFields(dir, Field{Path: []string{"language"}, Value: "es"}); err == nil {
		t.Fatal("expected an error for a non-mapping root")
	}

	after, err := os.ReadFile(filepath.Join(dir, FileName))
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Errorf("file was modified despite the error:\nbefore:\n%s\nafter:\n%s", before, after)
	}
}

// TestSetFieldsPreservesFileMode proves the on-disk permission bits survive
// the temp-file + rename write.
func TestSetFieldsPreservesFileMode(t *testing.T) {
	dir := t.TempDir()
	copyFixture(t, dir, "edit_comments.yaml", 0o640)

	if err := SetFields(dir, Field{Path: []string{"language"}, Value: "es"}); err != nil {
		t.Fatalf("SetFields: %v", err)
	}

	info, err := os.Stat(filepath.Join(dir, FileName))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o640 {
		t.Errorf("mode = %v, want 0640", info.Mode().Perm())
	}
}
