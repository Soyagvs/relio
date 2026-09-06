package releases

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/soyagvs/relio/internal/conventional"
	"github.com/soyagvs/relio/internal/gitrepo"
)

type fakeRepo struct {
	tags     []gitrepo.TagInfo
	messages map[string]string
	commits  map[string][]conventional.Raw // keyed by "to" tag name
	deleted  []string
}

func (f *fakeRepo) Tags() ([]gitrepo.TagInfo, error) { return f.tags, nil }

func (f *fakeRepo) TagMessage(name string) (string, error) { return f.messages[name], nil }

func (f *fakeRepo) CommitsBetween(from, to string) ([]conventional.Raw, error) {
	return f.commits[to], nil
}

func (f *fakeRepo) DeleteTag(name string) error {
	f.deleted = append(f.deleted, name)
	out := f.tags[:0]
	for _, t := range f.tags {
		if t.Name != name {
			out = append(out, t)
		}
	}
	f.tags = out
	return nil
}

func key(s string) tea.Msg {
	switch s {
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
	}
}

func send(m model, keys ...string) model {
	for _, k := range keys {
		next, _ := m.Update(key(k))
		m = next.(model)
	}
	return m
}

func setup(t *testing.T) (*fakeRepo, string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "CHANGELOG.md")
	content := "# Changelog\n\n" +
		"## [0.2.0] - 2026-09-06\n\n### Added\n\n- Second thing\n\n" +
		"## [0.1.0] - 2026-08-01\n\n### Added\n\n- First thing\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	fr := &fakeRepo{
		tags: []gitrepo.TagInfo{
			{Name: "v0.2.0", Date: "2026-09-06", DateTime: "2026-09-06 14:30", Subject: "release v0.2.0"},
			{Name: "v0.1.0", Date: "2026-08-01", DateTime: "2026-08-01 09:15", Subject: "release v0.1.0"},
		},
		messages: map[string]string{"v0.2.0": "release v0.2.0", "v0.1.0": "release v0.1.0"},
		commits: map[string][]conventional.Raw{
			"v0.2.0": {{Hash: "aaaaaaa0000000", Subject: "fix: second thing"}},
			"v0.1.0": {{Hash: "bbbbbbb0000000", Subject: "feat: first thing"}},
		},
	}
	return fr, path
}

func TestListsAllVersions(t *testing.T) {
	fr, path := setup(t)
	m := newModel(fr, "demo", path)
	if len(m.tags) != 2 {
		t.Fatalf("want 2 tags, got %d", len(m.tags))
	}
	v := m.View()
	if !strings.Contains(v, "v0.2.0") || !strings.Contains(v, "v0.1.0") {
		t.Errorf("view missing a version:\n%s", v)
	}
}

func TestDetailShowsNotesWithCommitHashForSelection(t *testing.T) {
	fr, path := setup(t)
	m := newModel(fr, "demo", path)

	v := m.View()
	if !strings.Contains(v, "Second thing") {
		t.Errorf("expected v0.2.0 notes in view:\n%s", v)
	}
	if !strings.Contains(v, "aaaaaaa") {
		t.Errorf("expected v0.2.0 commit hash next to its note:\n%s", v)
	}

	m = send(m, "down") // select v0.1.0
	v = m.View()
	if !strings.Contains(v, "First thing") || !strings.Contains(v, "bbbbbbb") {
		t.Errorf("expected v0.1.0 note + hash after moving down:\n%s", v)
	}
}

func TestDeleteRequiresConfirmation(t *testing.T) {
	fr, path := setup(t)
	m := newModel(fr, "demo", path)

	m = send(m, "d") // arm delete
	if m.mode != confirmDelete {
		t.Fatal("d should enter confirmDelete")
	}
	m = send(m, "n") // cancel
	if m.mode != browse || len(fr.deleted) != 0 {
		t.Fatalf("cancel failed: mode=%v deleted=%v", m.mode, fr.deleted)
	}

	m = send(m, "d", "y") // confirm
	if len(fr.deleted) != 1 || fr.deleted[0] != "v0.2.0" {
		t.Fatalf("expected v0.2.0 deleted, got %v", fr.deleted)
	}
	if len(m.tags) != 1 || m.tags[0].Name != "v0.1.0" {
		t.Errorf("tag list not refreshed: %+v", m.tags)
	}

	data, _ := os.ReadFile(path)
	if strings.Contains(string(data), "## [0.2.0]") || strings.Contains(string(data), "Second thing") {
		t.Errorf("changelog section not removed:\n%s", data)
	}
	if !strings.Contains(string(data), "## [0.1.0]") {
		t.Errorf("wrong section removed:\n%s", data)
	}
}

func TestQuitProducesStaticView(t *testing.T) {
	fr, path := setup(t)
	m := newModel(fr, "demo", path)
	m = send(m, "q")
	if !m.quit {
		t.Fatal("q should quit")
	}
	sv := m.View()
	if !strings.Contains(sv, "v0.2.0") || !strings.Contains(sv, "v0.1.0") {
		t.Errorf("static view should keep the list:\n%s", sv)
	}
}

func TestEnterPrintsSelectedVersionAndExits(t *testing.T) {
	fr, path := setup(t)
	m := newModel(fr, "demo", path)

	m = send(m, "down", "enter") // pick v0.1.0
	if !m.quit || m.picked != 1 {
		t.Fatalf("enter should quit with picked=1, got quit=%v picked=%d", m.quit, m.picked)
	}

	out := m.View() // this is what stays in the terminal
	for _, want := range []string{"demo -- release", "v0.1.0", "2026-08-01 09:15", "First thing", "bbbbbbb"} {
		if !strings.Contains(out, want) {
			t.Errorf("exit output missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "move") || strings.Contains(out, "delete") {
		t.Errorf("exit output should not carry the interactive footer:\n%s", out)
	}
}

func TestCursorClampsAfterDeletingLast(t *testing.T) {
	fr, path := setup(t)
	m := newModel(fr, "demo", path)
	m = send(m, "down") // v0.1.0, cursor=1
	m = send(m, "d", "y")
	if m.cursor != 0 {
		t.Errorf("cursor = %d, want 0 after deleting last row", m.cursor)
	}
}
