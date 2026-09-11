package settings

import (
	"os"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/soyagvs/relio/internal/config"
	"github.com/soyagvs/relio/internal/i18n"
	"github.com/soyagvs/relio/internal/userconfig"
)

func send(m model, keys ...string) model {
	for _, k := range keys {
		var msg tea.Msg
		switch k {
		case "up":
			msg = tea.KeyMsg{Type: tea.KeyUp}
		case "down":
			msg = tea.KeyMsg{Type: tea.KeyDown}
		case "enter":
			msg = tea.KeyMsg{Type: tea.KeyEnter}
		case " ":
			msg = tea.KeyMsg{Type: tea.KeySpace}
		case "esc":
			msg = tea.KeyMsg{Type: tea.KeyEsc}
		case "ctrl+c":
			msg = tea.KeyMsg{Type: tea.KeyCtrlC}
		default:
			msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
		}
		next, _ := m.Update(msg)
		m = next.(model)
	}
	return m
}

// withRepo writes a fresh .release.yaml at a temp dir and returns its root.
func withRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := config.Default("demo").Save(dir); err != nil {
		t.Fatalf("Save() = %v", err)
	}
	return dir
}

func TestCursorSkipsDisabledRowsOutsideRepo(t *testing.T) {
	m := newModel("")
	if m.rows[m.cursor].kind != rowRadio {
		t.Fatalf("initial cursor lands on kind %v, want rowRadio", m.rows[m.cursor].kind)
	}
	for range m.rows {
		m = send(m, "down")
		r := m.rows[m.cursor]
		if r.kind != rowRadio {
			t.Fatalf("cursor landed on a disabled row (kind %v) — outside a repo only the language radio rows are selectable", r.kind)
		}
	}
}

func TestCursorSkipsDisabledRowsAndReachesCheckboxesInsideRepo(t *testing.T) {
	dir := withRepo(t)
	m := newModel(dir)

	sawCheck := 0
	for range m.rows {
		m = send(m, "down")
		r := m.rows[m.cursor]
		if !r.enabled {
			t.Fatalf("cursor landed on a disabled row (kind %v)", r.kind)
		}
		if r.kind == rowCheck {
			sawCheck++
		}
	}
	if sawCheck == 0 {
		t.Fatal("cursor never reached a checkbox row inside a repo with .release.yaml")
	}
}

func TestSpaceOnRadioPersistsLanguage(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Cleanup(func() { i18n.SetLanguage("en") })
	i18n.SetLanguage("en")

	m := newModel("")
	// Move down until the cursor sits on the "es" radio row.
	for m.rows[m.cursor].value != "es" {
		m = send(m, "down")
	}
	m = send(m, " ")

	if i18n.Current() != "es" {
		t.Fatalf("i18n.Current() = %q, want es", i18n.Current())
	}
	if m.lang != "es" {
		t.Fatalf("model.lang = %q, want es", m.lang)
	}
	if got := userconfig.Load().Language; got != "es" {
		t.Fatalf("userconfig.Load().Language = %q, want es (persisted immediately)", got)
	}
	if m.err != "" {
		t.Fatalf("unexpected error: %q", m.err)
	}
}

func TestSpaceOnCheckTogglesAndPersists(t *testing.T) {
	dir := withRepo(t)
	m := newModel(dir)
	for m.rows[m.cursor].kind != rowCheck {
		m = send(m, "down")
	}
	before := m.checkValue(m.rows[m.cursor].field)
	m = send(m, " ")

	if m.err != "" {
		t.Fatalf("unexpected error: %q", m.err)
	}
	after := m.checkValue(m.rows[m.cursor].field)
	if after == before {
		t.Fatalf("checkValue unchanged after space: %v", after)
	}

	reloaded, err := config.Load(dir)
	if err != nil {
		t.Fatalf("config.Load() = %v", err)
	}
	if reloaded.Release.Contributors != after {
		t.Fatalf("disk contributors = %v, want %v (persisted immediately)", reloaded.Release.Contributors, after)
	}
}

func TestSpaceOnCheckTogglingOneDoesNotAffectTheOther(t *testing.T) {
	dir := withRepo(t)
	m := newModel(dir)
	var contributorsIdx, compareIdx = -1, -1
	for i, r := range m.rows {
		if r.kind != rowCheck {
			continue
		}
		if r.field[len(r.field)-1] == "contributors" {
			contributorsIdx = i
		}
		if r.field[len(r.field)-1] == "compare_link" {
			compareIdx = i
		}
	}
	if contributorsIdx == -1 || compareIdx == -1 {
		t.Fatal("expected both a contributors and a compare_link checkbox row")
	}

	beforeCompare := m.checkValue(m.rows[compareIdx].field)
	m.cursor = contributorsIdx
	m = send(m, " ")
	if got := m.checkValue(m.rows[compareIdx].field); got != beforeCompare {
		t.Fatalf("toggling contributors changed compare_link: %v -> %v", beforeCompare, got)
	}
}

func TestSpaceOnCheckRevertsOnPersistError(t *testing.T) {
	dir := withRepo(t)
	m := newModel(dir)
	for m.rows[m.cursor].kind != rowCheck {
		m = send(m, "down")
	}
	before := m.checkValue(m.rows[m.cursor].field)

	// Remove the config file after loading so the in-memory model still
	// believes the footer section is editable, but the persistence call
	// itself fails — this is the revert-on-error path.
	if err := os.Remove(config.Path(dir)); err != nil {
		t.Fatalf("os.Remove() = %v", err)
	}

	m = send(m, " ")
	if m.err == "" {
		t.Fatal("expected m.err to be set when SetFields fails")
	}
	if got := m.checkValue(m.rows[m.cursor].field); got != before {
		t.Fatalf("checkValue = %v after a failed persist, want reverted to %v", got, before)
	}
}

func TestViewRepaintsInNewLanguage(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Cleanup(func() { i18n.SetLanguage("en") })
	i18n.SetLanguage("en")

	m := newModel("")
	if !strings.Contains(m.View(), i18n.T(i18n.SettingsTitle)) {
		t.Fatalf("View() missing English title:\n%s", m.View())
	}

	for m.rows[m.cursor].value != "es" {
		m = send(m, "down")
	}
	m = send(m, " ")

	view := m.View()
	if !strings.Contains(view, i18n.T(i18n.SettingsTitle)) {
		t.Fatalf("View() after switching to es missing Spanish title:\n%s", view)
	}
}

func TestOutsideRepoFooterDisabledView(t *testing.T) {
	m := newModel("")
	if !strings.Contains(m.View(), i18n.T(i18n.SettingsFooterDisabledReason)) {
		t.Fatalf("View() outside a repo must show the disabled-footer reason:\n%s", m.View())
	}
}

func TestInsideRepoFooterHasNoDisabledReason(t *testing.T) {
	dir := withRepo(t)
	m := newModel(dir)
	if strings.Contains(m.View(), i18n.T(i18n.SettingsFooterDisabledReason)) {
		t.Fatalf("View() inside a repo with .release.yaml must not show the disabled-footer reason:\n%s", m.View())
	}
}

func TestQAndEscQuitWithoutKilling(t *testing.T) {
	for _, key := range []string{"q", "esc"} {
		m := send(newModel(""), key)
		if !m.done {
			t.Errorf("%q: done = false, want true", key)
		}
		if m.killed {
			t.Errorf("%q: killed = true, want false — q/esc is a soft return, not a hard quit", key)
		}
	}
}

func TestCtrlCIsHardQuit(t *testing.T) {
	m := send(newModel(""), "ctrl+c")
	if !m.done || !m.killed {
		t.Fatalf("ctrl+c: done=%v killed=%v, want both true", m.done, m.killed)
	}
}
