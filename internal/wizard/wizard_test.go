package wizard

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/soyagvs/relio/internal/config"
	"github.com/soyagvs/relio/internal/i18n"
	"github.com/soyagvs/relio/internal/release"
	"github.com/soyagvs/relio/internal/semver"
	"github.com/soyagvs/relio/internal/ui"
)

func basePlan() release.Plan {
	return release.Plan{
		Config:  config.Default("azeink"),
		Current: semver.Version{Major: 1, Minor: 3, Patch: 2, Prefix: "v"},
		Next:    semver.Version{Major: 1, Minor: 4, Patch: 0, Prefix: "v"},
		Bump:    semver.Minor,
	}
}

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
		default:
			msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
		}
		next, _ := m.Update(msg)
		m = next.(model)
	}
	return m
}

func TestConfirmDefault(t *testing.T) {
	m := send(newModel(basePlan()), "enter")
	if !m.result.Confirmed || m.result.Bump != semver.Minor {
		t.Errorf("result = %+v, want confirmed minor", m.result)
	}
}

func TestCancel(t *testing.T) {
	m := send(newModel(basePlan()), "q")
	if m.result.Confirmed {
		t.Errorf("expected not confirmed, got %+v", m.result)
	}
}

func TestOverrideToMajorThenConfirm(t *testing.T) {
	// down x3 -> "Change to major", enter applies it, enter again confirms.
	m := send(newModel(basePlan()), "down", "down", "down", "enter", "enter")
	if !m.result.Confirmed || m.result.Bump != semver.Major {
		t.Errorf("result = %+v, want confirmed major", m.result)
	}
	if m.nextVersion().String() != "v2.0.0" {
		t.Errorf("nextVersion = %s, want v2.0.0", m.nextVersion())
	}
}

func TestNextVersionShowsPlanNextForPrerelease(t *testing.T) {
	p := release.Plan{
		Config:     config.Default("azeink"),
		Current:    semver.Version{Major: 1, Minor: 5, Prefix: "v"},
		Next:       semver.Version{Major: 1, Minor: 6, Prefix: "v", Pre: "rc.1"},
		Bump:       semver.Minor,
		Prerelease: true,
	}
	m := newModel(p)
	if got := m.nextVersion().String(); got != "v1.6.0-rc.1" {
		t.Errorf("nextVersion = %s, want v1.6.0-rc.1 (plan.Next verbatim)", got)
	}
}

func TestYKeyConfirmsImmediately(t *testing.T) {
	m := send(newModel(basePlan()), "y")
	if !m.result.Confirmed {
		t.Errorf("y should confirm, got %+v", m.result)
	}
}

func TestCursorClamps(t *testing.T) {
	m := send(newModel(basePlan()), "up", "up", "up")
	if m.cursor != 0 {
		t.Errorf("cursor = %d, want 0", m.cursor)
	}
	m = send(m, "down", "down", "down", "down", "down", "down")
	if m.cursor != len(m.choices)-1 {
		t.Errorf("cursor = %d, want %d", m.cursor, len(m.choices)-1)
	}
}

// TestGoldenEnglishDefaultUnchanged pins wizard.go's own chrome against
// i18n.T() under the default "en" language: converting wizard.go's literals
// to i18n.T() calls MUST NOT change a single byte of English output. This is
// the RED/refactor safety net for the wizard.go -> i18n.T() conversion
// (slice 5a).
func TestGoldenEnglishDefaultUnchanged(t *testing.T) {
	prev := i18n.Current()
	if prev != "en" {
		i18n.SetLanguage("en")
	}
	t.Cleanup(func() { i18n.SetLanguage(prev) })

	p := basePlan()

	// Header + bump-from line, byte-identical to the historical hardcoded
	// strings.
	m := newModel(p)
	wantHeader := ui.Key.Render("Release v1.4.0") +
		ui.Dim.Render("   (minor from v1.3.2)") + "\n\n"
	v := m.View()
	if !strings.HasPrefix(v, wantHeader) {
		t.Errorf("View() header = %q, want prefix %q", v, wantHeader)
	}

	// Choice labels, byte-identical to the historical hardcoded strings.
	for _, want := range []string{"Change to patch", "Change to minor", "Change to major", "Cancel"} {
		if !strings.Contains(v, want) {
			t.Errorf("View() missing choice label %q:\n%s", want, v)
		}
	}
	if !strings.Contains(v, "Create v1.4.0") {
		t.Errorf("View() missing formatted confirm choice %q:\n%s", "Create v1.4.0", v)
	}

	// Footer hint, byte-identical to the historical hardcoded string.
	wantHint := "↑/↓ move · enter select · y confirm · q cancel"
	if got := i18n.T(i18n.WizardHint); got != wantHint {
		t.Errorf("i18n.T(WizardHint) under en = %q, want %q", got, wantHint)
	}
	if !strings.HasSuffix(v, wantHint) {
		t.Errorf("View() does not end with the expected hint line:\n%s", v)
	}

	// Confirmed trace line, byte-identical to the historical hardcoded
	// string.
	confirmed := send(newModel(p), "enter")
	wantConfirmed := ui.Dim.Render("→ confirmed v1.4.0") + "\n"
	if got := confirmed.View(); got != wantConfirmed {
		t.Errorf("confirmed View() = %q, want %q", got, wantConfirmed)
	}

	// Cancelled trace line, byte-identical to the historical hardcoded
	// string.
	cancelled := send(newModel(p), "q")
	wantCancelled := ui.Dim.Render("→ cancelled") + "\n"
	if got := cancelled.View(); got != wantCancelled {
		t.Errorf("cancelled View() = %q, want %q", got, wantCancelled)
	}
}

// TestChoiceLabelsLocalizeUnderSpanish proves the wizard's choice labels,
// header, bump-from line, hint, and trace lines actually route through
// i18n.T (not just accidentally identical in English): switching to "es"
// must change every one of them and must NOT leave the English literal
// behind.
func TestChoiceLabelsLocalizeUnderSpanish(t *testing.T) {
	prev := i18n.Current()
	i18n.SetLanguage("es")
	t.Cleanup(func() { i18n.SetLanguage(prev) })

	p := basePlan()
	v := newModel(p).View()

	for _, english := range []string{"Change to patch", "Change to minor", "Change to major", "Cancel\n"} {
		if strings.Contains(v, english) {
			t.Errorf("es View() still contains English literal %q:\n%s", english, v)
		}
	}
	for _, spanish := range []string{"Cambiar a patch", "Cambiar a minor", "Cambiar a major", "Cancelar"} {
		if !strings.Contains(v, spanish) {
			t.Errorf("es View() missing Spanish %q:\n%s", spanish, v)
		}
	}
	if !strings.Contains(v, "Lanzamiento v1.4.0") {
		t.Errorf("es View() missing localized header, got:\n%s", v)
	}
	if !strings.Contains(v, "(minor desde v1.3.2)") {
		t.Errorf("es View() missing localized bump-from line, got:\n%s", v)
	}
	if !strings.HasSuffix(v, "↑/↓ mover · enter seleccionar · y confirmar · q cancelar") {
		t.Errorf("es View() missing localized hint, got:\n%s", v)
	}

	confirmed := send(newModel(p), "enter")
	if got := confirmed.View(); !strings.Contains(got, "confirmado v1.4.0") || strings.Contains(got, "confirmed") {
		t.Errorf("es confirmed View() = %q, want localized confirmation", got)
	}

	cancelled := send(newModel(p), "q")
	if got := cancelled.View(); !strings.Contains(got, "cancelado") || strings.Contains(got, "cancelled") {
		t.Errorf("es cancelled View() = %q, want localized cancellation", got)
	}
}
