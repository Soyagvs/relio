package pick

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
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
