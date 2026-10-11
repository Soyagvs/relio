package releases

import (
	"errors"
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
	tags      []gitrepo.TagInfo
	messages  map[string]string
	commits   map[string][]conventional.Raw // keyed by "to" tag name
	deleted   []string
	deleteErr error // when set, DeleteTag returns this instead of mutating tags
}

func (f *fakeRepo) Tags() ([]gitrepo.TagInfo, error) { return f.tags, nil }

func (f *fakeRepo) TagMessage(name string) (string, error) { return f.messages[name], nil }

func (f *fakeRepo) CommitsBetween(from, to string) ([]conventional.Raw, error) {
	return f.commits[to], nil
}

func (f *fakeRepo) DeleteTag(name string) error {
	if f.deleteErr != nil {
		return f.deleteErr
	}
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

func TestPaginatedTableShowsReleaseMetadata(t *testing.T) {
	fr, path := setup(t)
	m := newModel(fr, "demo", path)

	v := m.View()
	for _, want := range []string{"Commits", "Version", "Date", "1", "v0.2.0", "2026-09-06", "Page 1/1"} {
		if !strings.Contains(v, want) {
			t.Fatalf("view missing %q:\n%s", want, v)
		}
	}
}

func TestPageNavigationKeepsPositionManageable(t *testing.T) {
	fr, path := setup(t)
	for i := 3; i <= 12; i++ {
		name := fmt.Sprintf("v0.%d.0", i)
		fr.tags = append(fr.tags, gitrepo.TagInfo{Name: name, Date: "2026-09-01", Subject: "release " + name})
		fr.commits[name] = []conventional.Raw{{Hash: "ccccccc0000000", Subject: "fix: item"}}
	}
	m := newModel(fr, "demo", path)

	m = send(m, "n")
	if m.cursor != releasesPageSize {
		t.Fatalf("n should move to next page first row, cursor=%d", m.cursor)
	}
	v := m.View()
	if !strings.Contains(v, "Page 2/2") || strings.Contains(v, "v0.2.0") {
		t.Fatalf("view should show only page 2 releases:\n%s", v)
	}

	m = send(m, "p")
	if m.cursor != 0 || !strings.Contains(m.View(), "Page 1/2") {
		t.Fatalf("p should return to first page, cursor=%d view=\n%s", m.cursor, m.View())
	}
}

func TestDetailShowsNotesWithCommitHashForSelection(t *testing.T) {
	fr, path := setup(t)
	m := newModel(fr, "demo", path)

	m = send(m, "enter")
	v := m.View()
	if !strings.Contains(v, "Second thing") {
		t.Errorf("expected v0.2.0 notes in preview:\n%s", v)
	}
	if !strings.Contains(v, "aaaaaaa") {
		t.Errorf("expected v0.2.0 commit hash next to its note:\n%s", v)
	}

	m = send(m, "esc", "down", "enter") // select v0.1.0 and preview
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

func TestQuitLeavesNoStaticReleaseList(t *testing.T) {
	fr, path := setup(t)
	m := newModel(fr, "demo", path)
	m = send(m, "q")
	if !m.quit {
		t.Fatal("q should quit")
	}
	if sv := m.View(); sv != "" {
		t.Errorf("q should return to menu without leaving release list behind, got:\n%s", sv)
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

func TestBackRowExitsWithoutPickingRelease(t *testing.T) {
	fr, path := setup(t)
	m := newModel(fr, "demo", path)

	view := m.View()
	if !strings.Contains(view, backHomeLabel) {
		t.Fatalf("view missing home back row %q:\n%s", backHomeLabel, view)
	}

	m = send(m, "down", "down")
	if m.cursor != len(m.tags) || !m.onBack() {
		t.Fatalf("cursor should move to back row, got cursor=%d tags=%d", m.cursor, len(m.tags))
	}

	m = send(m, "enter")
	if !m.quit || m.killed || m.picked != -1 {
		t.Fatalf("back row should soft-exit without picking, got quit=%v killed=%v picked=%d", m.quit, m.killed, m.picked)
	}
}

func TestBackRowNavigationReturnsToLastRelease(t *testing.T) {
	fr, path := setup(t)
	m := newModel(fr, "demo", path)

	m = send(m, "down", "down", "up")
	if m.cursor != len(m.tags)-1 {
		t.Fatalf("up from back row should return to last release, got cursor=%d", m.cursor)
	}

	m = send(m, "l")
	if m.quit || m.mode != preview {
		t.Fatalf("l on release should open preview, got quit=%v mode=%v", m.quit, m.mode)
	}
}

func TestEmptyReleasesShowSelectableBack(t *testing.T) {
	empty := &fakeRepo{}
	m := newModel(empty, "demo", filepath.Join(t.TempDir(), "CHANGELOG.md"))

	view := m.View()
	if !strings.Contains(view, backHomeLabel) || !m.onBack() {
		t.Fatalf("empty view should show selectable back row, onBack=%v:\n%s", m.onBack(), view)
	}

	m = send(m, "enter")
	if !m.quit || m.killed || m.picked != -1 {
		t.Fatalf("empty back row should soft-exit without picking, got quit=%v killed=%v picked=%d", m.quit, m.killed, m.picked)
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

// TestDoDeleteTagFailureLeavesTagsUntouched is the model-level regression
// counterpart to TestDeleteTagFailurePropagatesRawError (delete_test.go): a
// failure deleting the tag surfaces as m.err (the raw error text, matching
// the pre-extraction behavior) and never removes the tag from the list.
func TestDoDeleteTagFailureLeavesTagsUntouched(t *testing.T) {
	fr, path := setup(t)
	fr.deleteErr = errors.New("boom")
	m := newModel(fr, "demo", path)

	m = send(m, "d", "y")
	if m.err != "boom" {
		t.Errorf("m.err = %q, want %q", m.err, "boom")
	}
	if len(m.tags) != 2 {
		t.Errorf("tags = %v, want unchanged (2) after a failed delete", m.tags)
	}
	if len(fr.deleted) != 0 {
		t.Errorf("deleted = %v, want none recorded on failure", fr.deleted)
	}
}

// TestDoDeleteReloadsAfterChangelogReadError guards against the tag list
// going stale in the TUI: when the tag is deleted successfully but the
// subsequent changelog read fails for a reason other than "file does not
// exist" (e.g. the changelog path is a directory), doDelete must still
// reload so the browser stops showing the already-deleted tag as present.
func TestDoDeleteReloadsAfterChangelogReadError(t *testing.T) {
	fr := &fakeRepo{
		tags: []gitrepo.TagInfo{
			{Name: "v0.2.0", Date: "2026-09-06"},
			{Name: "v0.1.0", Date: "2026-08-01"},
		},
	}
	dir := t.TempDir()
	// A directory at the changelog path makes the post-delete read fail with
	// an error that is not os.IsNotExist.
	asDir := filepath.Join(dir, "CHANGELOG.md")
	if err := os.Mkdir(asDir, 0o755); err != nil {
		t.Fatal(err)
	}
	m := newModel(fr, "demo", asDir)

	m = send(m, "d", "y")
	if len(fr.deleted) != 1 || fr.deleted[0] != "v0.2.0" {
		t.Fatalf("expected v0.2.0 deleted, got %v", fr.deleted)
	}
	if m.err == "" {
		t.Error("m.err should be set when the post-delete changelog read fails")
	}
	if len(m.tags) != 1 || m.tags[0].Name != "v0.1.0" {
		t.Errorf("tags = %+v, want reloaded to just [v0.1.0] -- the tag is already gone even though the changelog step failed", m.tags)
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

func TestLShowsPreviewAndHReturnsToTable(t *testing.T) {
	fr, path := setup(t)
	m := newModel(fr, "demo", path)

	m = send(m, "down", "l")
	if m.quit || m.mode != preview || m.detailOffset != 0 {
		t.Fatalf("l should open preview without quitting, got quit=%v mode=%v offset=%d", m.quit, m.mode, m.detailOffset)
	}
	if v := m.View(); !strings.Contains(v, "Preview") || !strings.Contains(v, "First thing") || !strings.Contains(v, "│") {
		t.Fatalf("preview should show selected release notes:\n%s", v)
	}

	m = send(m, "h")
	if m.quit || m.mode != browse {
		t.Fatalf("h from preview should return to table, got quit=%v mode=%v", m.quit, m.mode)
	}
}

func TestSelectModelConfirmsOnlyFromPreview(t *testing.T) {
	fr, path := setup(t)
	m := newSelectModel(fr, "demo", path)

	m = send(m, "enter")
	if m.confirmed || m.mode != preview {
		t.Fatalf("first enter should open preview, got confirmed=%v mode=%v", m.confirmed, m.mode)
	}

	m = send(m, "esc")
	if m.confirmed || m.mode != browse {
		t.Fatalf("esc should go back to table, got confirmed=%v mode=%v", m.confirmed, m.mode)
	}

	m = send(m, "enter", "down")
	if !m.previewConfirm || m.previewBack {
		t.Fatalf("down from preview should focus Confirm, confirm=%v back=%v", m.previewConfirm, m.previewBack)
	}
	if v := m.View(); !strings.Contains(v, "▸ "+ui.Key.Render("✔ confirm")) {
		t.Fatalf("preview should render focused Confirm button:\n%s", v)
	}
	m = send(m, "enter")
	if !m.confirmed {
		t.Fatal("enter on Confirm should confirm selected release")
	}
	if got := m.View(); got != "" {
		t.Fatalf("confirmed selector should leave no preview in scrollback, got:\n%s", got)
	}
}

func TestSelectableModeCannotDeleteReleases(t *testing.T) {
	fr, path := setup(t)
	m := newSelectModel(fr, "demo", path)

	m = send(m, "d", "y")
	if m.mode == confirmDelete || len(fr.deleted) != 0 {
		t.Fatalf("selectable release picker must not delete releases, mode=%v deleted=%v", m.mode, fr.deleted)
	}
	if v := m.View(); strings.Contains(v, keyHint("d", i18n.T(i18n.ReleasesHintDelete))) {
		t.Fatalf("selectable release picker should not advertise delete:\n%s", v)
	}
}

func TestListModelEnterPreviewDoesNotConfirm(t *testing.T) {
	fr, path := setup(t)
	m := newModel(fr, "demo", path)

	m = send(m, "enter", "enter")
	if m.confirmed || m.quit {
		t.Fatalf("list releases may view but not confirm, got confirmed=%v quit=%v", m.confirmed, m.quit)
	}
}

func TestDetailPaneScrollsLongNotes(t *testing.T) {
	fr, path := setup(t)
	many := make([]conventional.Raw, 0, detailPaneLines+4)
	for i := 0; i < detailPaneLines+4; i++ {
		many = append(many, conventional.Raw{Hash: fmt.Sprintf("%07d0000000", i), Subject: fmt.Sprintf("fix: scroll item %02d", i)})
	}
	fr.commits["v0.2.0"] = many
	m := newModel(fr, "demo", path)
	m = send(m, "l")
	before := m.View()
	if !strings.Contains(before, "Scroll item 00") || !strings.Contains(before, "↓ more") {
		t.Fatalf("detail pane should start at top with more indicator:\n%s", before)
	}

	m = send(m, "down", "down", "down")
	if m.detailOffset != 3 {
		t.Fatalf("detail offset = %d, want 3", m.detailOffset)
	}
	after := m.View()
	if !strings.Contains(after, "↑ more") || strings.Contains(after, "relio -- release") {
		t.Fatalf("detail pane should scroll down and show up indicator:\n%s", after)
	}
}

func TestPreviewCanMoveDownToBackRow(t *testing.T) {
	fr, path := setup(t)
	m := newModel(fr, "demo", path)
	m = send(m, "enter", "down")
	if !m.previewBack {
		t.Fatalf("down at end of preview should focus Back row")
	}
	if v := m.View(); !strings.Contains(v, "▸ "+ui.Key.Render(backLabel)) {
		t.Fatalf("preview should render focused Back row:\n%s", v)
	}

	m = send(m, "enter")
	if m.mode != browse || m.previewBack {
		t.Fatalf("enter on preview Back should return to table, mode=%v previewBack=%v", m.mode, m.previewBack)
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

	// Footer hints, byte-identical to the current English catalog strings.
	wantFooter := keyHint("↑/↓", "move") + keyHint("enter", "details") + keyHint("n/p", "page") +
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
	wantEmpty := wantTitle + ui.Dim.Render("No releases yet. Create one from the menu.") + "\n\n" +
		"▸ " + ui.Key.Render(backHomeLabel) + "\n\n" +
		keyHint("↑/↓", "move") + keyHint("enter", "details") + keyHint("n/p", "page") + keyHint("d", "delete") + keyHint("q", "back")
	if ev := em.View(); ev != wantEmpty {
		t.Errorf("empty View() = %q, want %q", ev, wantEmpty)
	}

	quitEmpty := send(em, "q")
	if sv := quitEmpty.View(); sv != "" {
		t.Errorf("empty staticView() = %q, want blank", sv)
	}

	// notesFor's "(no notes for this version)" fallback, byte-identical —
	// this exercises only the fallback placeholder, never
	// ui.Notes/ui.ReleaseText.
	noNotes := &fakeRepo{tags: []gitrepo.TagInfo{{Name: "v0.1.0", Date: "2026-08-01"}}}
	nm := newModel(noNotes, "demo", filepath.Join(t.TempDir(), "CHANGELOG.md"))
	wantNoNotes := ui.Dim.Render("(no notes for this version)")
	if nv := send(nm, "enter").View(); !strings.Contains(nv, wantNoNotes) {
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
	if strings.Contains(v, "print notes & exit") || strings.Contains(v, "delete release") || strings.Contains(v, "enter details") {
		t.Errorf("es View() still contains English footer hints:\n%s", v)
	}
	for _, spanish := range []string{"mover", "detalles", "eliminar"} {
		if !strings.Contains(v, spanish) {
			t.Errorf("es View() missing localized hint %q, got:\n%s", spanish, v)
		}
	}
	if !strings.Contains(v, backHomeLabel) || !strings.Contains(v, "q volver") {
		t.Errorf("es View() should show the explicit back row instead of q-back hint, got:\n%s", v)
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
	if !strings.Contains(ev, backHomeLabel) || !strings.Contains(ev, "q volver") {
		t.Errorf("es empty View() should show the explicit back row instead of q-back hint, got:\n%s", ev)
	}

	quitEmpty := send(em, "q")
	if sv := quitEmpty.View(); sv != "" {
		t.Errorf("es staticView() = %q, want blank", sv)
	}

	noNotes := &fakeRepo{tags: []gitrepo.TagInfo{{Name: "v0.1.0", Date: "2026-08-01"}}}
	nm := newModel(noNotes, "demo", filepath.Join(t.TempDir(), "CHANGELOG.md"))
	if nv := send(nm, "enter").View(); !strings.Contains(nv, "(sin notas para esta versión)") || strings.Contains(nv, "no notes for this version") {
		t.Errorf("es View() missing localized no-notes fallback, got:\n%s", nv)
	}
}
