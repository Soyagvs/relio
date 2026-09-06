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
	if m.chosen || !m.quit {
		t.Errorf("q should cancel: chosen=%v quit=%v", m.chosen, m.quit)
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
