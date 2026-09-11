package pick

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/soyagvs/relio/internal/i18n"
	"github.com/soyagvs/relio/internal/ui"
)

var sample = []Item{
	{Label: "Minimal", Desc: "short", Value: "minimal"},
	{Label: "Technical", Desc: "terse", Value: "technical"},
	{Label: "Casual", Desc: "loose", Value: "casual"},
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

func TestSelectsValue(t *testing.T) {
	m := send(model{title: "t", items: sample}, "down", "enter")
	if !m.chosen || m.items[m.cursor].Value != "technical" {
		t.Errorf("got chosen=%v value=%q", m.chosen, m.items[m.cursor].Value)
	}
}

func TestCancel(t *testing.T) {
	m := send(model{title: "t", items: sample}, "q")
	if m.chosen || !m.quit || m.killed {
		t.Errorf("q should back out (not kill): chosen=%v quit=%v killed=%v", m.chosen, m.quit, m.killed)
	}
}

func TestCtrlCKills(t *testing.T) {
	m := send(model{title: "t", items: sample}, "ctrl+c")
	if !m.killed || m.chosen {
		t.Errorf("ctrl+c should hard-quit: chosen=%v killed=%v", m.chosen, m.killed)
	}
}

func TestCursorClamps(t *testing.T) {
	m := send(model{title: "t", items: sample}, "up", "up", "up")
	if m.cursor != 0 {
		t.Errorf("cursor = %d, want 0", m.cursor)
	}
	m = send(m, "down", "down", "down", "down")
	if m.cursor != len(sample)-1 {
		t.Errorf("cursor = %d, want %d", m.cursor, len(sample)-1)
	}
}

func TestViewListsOptions(t *testing.T) {
	v := model{title: "Pick one", items: sample}.View()
	for _, it := range sample {
		if !strings.Contains(v, it.Label) {
			t.Errorf("view missing %q:\n%s", it.Label, v)
		}
	}
}

// TestGoldenEnglishDefaultUnchanged pins pick.go's own chrome (title
// rendering aside, which is caller-supplied) against i18n.T() under the
// default "en" language: converting pick.go's literals to i18n.T() calls
// MUST NOT change a single byte of English output. This is the RED/refactor
// safety net for the pick.go -> i18n.T() conversion (slice 4a).
func TestGoldenEnglishDefaultUnchanged(t *testing.T) {
	prev := i18n.Current()
	if prev != "en" {
		i18n.SetLanguage("en")
	}
	t.Cleanup(func() { i18n.SetLanguage(prev) })

	// Footer hint line, byte-identical to the historical hardcoded string.
	wantHint := "↑/↓ move · enter select · q cancel"
	if got := i18n.T(i18n.PickHint); got != wantHint {
		t.Errorf("i18n.T(PickHint) under en = %q, want %q", got, wantHint)
	}
	v := model{title: "Pick one", items: sample}.View()
	if !strings.HasSuffix(v, wantHint) {
		t.Errorf("View() does not end with the expected hint line:\n%s", v)
	}

	// Cancelled trace line, byte-identical to the historical hardcoded string.
	wantCancelled := "→ cancelled"
	if got := i18n.T(i18n.PickCancelled); got != wantCancelled {
		t.Errorf("i18n.T(PickCancelled) under en = %q, want %q", got, wantCancelled)
	}
	cancelled := send(model{title: "t", items: sample}, "q")
	wantView := ui.Dim.Render(wantCancelled) + "\n"
	if got := cancelled.View(); got != wantView {
		t.Errorf("cancelled View() = %q, want %q", got, wantView)
	}
}
