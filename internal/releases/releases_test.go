package releases

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/soyagvs/relio/internal/conventional"
	"github.com/soyagvs/relio/internal/gitrepo"
	"github.com/soyagvs/relio/internal/i18n"
	"github.com/soyagvs/relio/internal/ui"
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
	case "ctrl+c":
		return tea.KeyMsg{Type: tea.KeyCtrlC}
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

func TestQBacksOutWithoutKilling(t *testing.T) {
	fr, path := setup(t)
	m := newModel(fr, "demo", path)
	m = send(m, "q")
	if !m.quit {
		t.Fatal("q should set quit")
	}
	if m.killed {
		t.Error("q should not set killed — it is a soft back-out, not a hard quit")
	}
}

func TestCtrlCKills(t *testing.T) {
	fr, path := setup(t)
	m := newModel(fr, "demo", path)
	m = send(m, "ctrl+c")
	if !m.quit || !m.killed {
		t.Fatalf("ctrl+c should set quit and killed, got quit=%v killed=%v", m.quit, m.killed)
	}
}

func TestCtrlCKillsFromConfirmDelete(t *testing.T) {
	fr, path := setup(t)
	m := newModel(fr, "demo", path)
	m = send(m, "d") // arm delete
	if m.mode != confirmDelete {
		t.Fatal("d should enter confirmDelete")
	}
	m = send(m, "ctrl+c")
	if !m.killed {
		t.Errorf("ctrl+c from the delete prompt should hard-quit (killed=true), got killed=%v", m.killed)
	}
	if len(fr.deleted) != 0 {
		t.Errorf("ctrl+c must not delete anything, got %v", fr.deleted)
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
	for _, want := range []string{"relio -- release", "demo · v0.1.0", "2026-08-01 · 09:15 · 1 commits", "First thing", "bbbbbbb"} {
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

// TestGoldenEnglishDefaultUnchanged pins releases.go's own chrome against
// i18n.T() under the default "en" language: converting releases.go's
// literals to i18n.T() calls MUST NOT change a single byte of English
// output. This is the RED/refactor safety net for the releases.go ->
// i18n.T() conversion (slice 5b). It intentionally never exercises
// notesFor's ui.Notes/ui.ReleaseText path — that content stays English by
// a separate, permanent contract enforced by
// internal/ui/artifact_invariance_test.go, not by this test.
func TestGoldenEnglishDefaultUnchanged(t *testing.T) {
	prev := i18n.Current()
	if prev != "en" {
		i18n.SetLanguage("en")
	}
	t.Cleanup(func() { i18n.SetLanguage(prev) })

	fr, path := setup(t)
	m := newModel(fr, "demo", path)

	// Title, byte-identical to the historical hardcoded string.
	wantTitle := ui.Title.Render("⬢ Releases") + "\n\n"
	v := m.View()
	if !strings.HasPrefix(v, wantTitle) {
		t.Errorf("View() title = %q, want prefix %q", v, wantTitle)
	}

	// Footer hints, byte-identical to the historical hardcoded strings.
	wantEnterHint := ui.Key.Render(fmt.Sprintf("%-7s", "enter")) + ui.Dim.Render("print notes & exit")
	if !strings.Contains(v, wantEnterHint) {
		t.Errorf("View() missing enter hint %q:\n%s", wantEnterHint, v)
	}
	wantDeleteHint := ui.Key.Render(fmt.Sprintf("%-7s", "d")) + ui.Dim.Render("delete release")
	if !strings.Contains(v, wantDeleteHint) {
		t.Errorf("View() missing delete hint %q:\n%s", wantDeleteHint, v)
	}
	wantFooter := keyHint("↑/↓", "move") + keyHint("enter", "show & exit") +
		keyHint("d", "delete") + keyHint("q", "back")
	if !strings.HasSuffix(v, wantFooter) {
		t.Errorf("View() does not end with the expected footer:\n%s", v)
	}

	// Delete-confirmation prompt, byte-identical to the historical hardcoded
	// string.
	confirming := send(m, "d")
	wantConfirm := "\n" + ui.Warn.Render("Delete v0.2.0? This removes the git tag and its changelog section.  [y/N]")
	if cv := confirming.View(); !strings.HasSuffix(cv, wantConfirm) {
		t.Errorf("confirmDelete View() = %q, want suffix %q", cv, wantConfirm)
	}

	// Delete-cancelled status, byte-identical to the historical hardcoded
	// string.
	fr2, path2 := setup(t)
	m2 := newModel(fr2, "demo", path2)
	cancelled := send(m2, "d", "n")
	if cv := cancelled.View(); !strings.Contains(cv, "Delete cancelled.") {
		t.Errorf("View() after cancel missing %q:\n%s", "Delete cancelled.", cv)
	}

	// Deleted status (tag + changelog section removed), byte-identical to
	// the historical hardcoded string.
	fr3, path3 := setup(t)
	m3 := newModel(fr3, "demo", path3)
	deleted := send(m3, "d", "y")
	if dv := deleted.View(); !strings.Contains(dv, "Deleted v0.2.0 (tag + changelog section).") {
		t.Errorf("View() after delete missing %q:\n%s", "Deleted v0.2.0 (tag + changelog section).", dv)
	}

	// Empty-state chrome, byte-identical to the historical hardcoded
	// strings.
	empty := &fakeRepo{}
	em := newModel(empty, "demo", filepath.Join(t.TempDir(), "CHANGELOG.md"))
	wantEmpty := wantTitle + ui.Dim.Render("No releases yet. Create one from the menu.") + "\n\n" + ui.Dim.Render("q back")
	if ev := em.View(); ev != wantEmpty {
		t.Errorf("empty View() = %q, want %q", ev, wantEmpty)
	}

	// staticView "(none)" placeholder, byte-identical.
	quitEmpty := send(em, "q")
	wantNone := ui.Title.Render("⬢ Releases") + "\n" + ui.Dim.Render("  (none)") + "\n"
	if sv := quitEmpty.View(); sv != wantNone {
		t.Errorf("empty staticView() = %q, want %q", sv, wantNone)
	}

	// notesFor's "(no notes for this version)" fallback, byte-identical —
	// this exercises only the fallback placeholder, never
	// ui.Notes/ui.ReleaseText.
	noNotes := &fakeRepo{tags: []gitrepo.TagInfo{{Name: "v0.1.0", Date: "2026-08-01"}}}
	nm := newModel(noNotes, "demo", filepath.Join(t.TempDir(), "CHANGELOG.md"))
	wantNoNotes := ui.Dim.Render("(no notes for this version)")
	if nv := nm.View(); !strings.Contains(nv, wantNoNotes) {
		t.Errorf("View() missing no-notes fallback %q:\n%s", wantNoNotes, nv)
	}
}

// TestChromeLocalizesUnderSpanish proves releases.go's chrome — title,
// empty-state message, footer hints, delete-confirmation prompt, delete
// outcome status lines, and the no-notes fallback — actually routes
// through i18n.T (not just accidentally identical in English): switching
// to "es" must change every one of them and must never leak the English
// literal. notesFor's ui.Notes/ui.ReleaseText path is deliberately NOT
// exercised for divergence here — that boundary is asserted invariant, not
// localized, by internal/ui/artifact_invariance_test.go.
func TestChromeLocalizesUnderSpanish(t *testing.T) {
	prev := i18n.Current()
	i18n.SetLanguage("es")
	t.Cleanup(func() { i18n.SetLanguage(prev) })

	fr, path := setup(t)
	m := newModel(fr, "demo", path)
	v := m.View()

	if !strings.Contains(v, "Lanzamientos") {
		t.Errorf("es View() missing localized title, got:\n%s", v)
	}
	if strings.Contains(v, "print notes & exit") || strings.Contains(v, "delete release") {
		t.Errorf("es View() still contains English footer hints:\n%s", v)
	}
	for _, spanish := range []string{"mostrar notas y salir", "eliminar lanzamiento", "mover", "mostrar y salir", "eliminar", "volver"} {
		if !strings.Contains(v, spanish) {
			t.Errorf("es View() missing localized hint %q, got:\n%s", spanish, v)
		}
	}

	confirming := send(m, "d")
	cv := confirming.View()
	if strings.Contains(cv, "This removes the git tag") {
		t.Errorf("es confirmDelete View() still contains English, got:\n%s", cv)
	}
	if !strings.Contains(cv, "¿Eliminar v0.2.0? Esto elimina la etiqueta de git y su sección del historial de cambios.") {
		t.Errorf("es confirmDelete View() missing localized prompt, got:\n%s", cv)
	}

	fr2, path2 := setup(t)
	m2 := newModel(fr2, "demo", path2)
	cancelled := send(m2, "d", "n")
	if cv := cancelled.View(); !strings.Contains(cv, "Eliminación cancelada.") || strings.Contains(cv, "Delete cancelled.") {
		t.Errorf("es View() after cancel = %q, want localized cancellation", cv)
	}

	fr3, path3 := setup(t)
	m3 := newModel(fr3, "demo", path3)
	deleted := send(m3, "d", "y")
	if dv := deleted.View(); !strings.Contains(dv, "Eliminado v0.2.0 (etiqueta + sección del historial de cambios).") || strings.Contains(dv, "Deleted") {
		t.Errorf("es View() after delete = %q, want localized deletion", dv)
	}

	empty := &fakeRepo{}
	em := newModel(empty, "demo", filepath.Join(t.TempDir(), "CHANGELOG.md"))
	ev := em.View()
	if !strings.Contains(ev, "Aún no hay lanzamientos. Crea uno desde el menú.") || strings.Contains(ev, "No releases yet") {
		t.Errorf("es empty View() = %q, want localized empty state", ev)
	}
	if !strings.Contains(ev, "q volver") {
		t.Errorf("es empty View() missing localized back hint, got:\n%s", ev)
	}

	quitEmpty := send(em, "q")
	if sv := quitEmpty.View(); !strings.Contains(sv, "(ninguno)") || strings.Contains(sv, "(none)") {
		t.Errorf("es staticView() = %q, want localized none placeholder", sv)
	}

	noNotes := &fakeRepo{tags: []gitrepo.TagInfo{{Name: "v0.1.0", Date: "2026-08-01"}}}
	nm := newModel(noNotes, "demo", filepath.Join(t.TempDir(), "CHANGELOG.md"))
	if nv := nm.View(); !strings.Contains(nv, "(sin notas para esta versión)") || strings.Contains(nv, "no notes for this version") {
		t.Errorf("es View() missing localized no-notes fallback, got:\n%s", nv)
	}
}
