package menu

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
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
		default:
			msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
		}
		next, _ := m.Update(msg)
		m = next.(model)
	}
	return m
}

func TestSelectCreateRelease(t *testing.T) {
	m := send(model{}, "enter")
	if m.result != CreateRelease {
		t.Errorf("result = %v, want CreateRelease", m.result)
	}
}

func TestSelectAuth(t *testing.T) {
	m := send(model{}, "down", "enter")
	if m.result != GitHubAuth {
		t.Errorf("result = %v, want GitHubAuth", m.result)
	}
}

func TestSelectExit(t *testing.T) {
	m := send(model{}, "down", "down", "enter")
	if m.result != Exit {
		t.Errorf("result = %v, want Exit", m.result)
	}
}

func TestQuitKeyIsExit(t *testing.T) {
	m := send(model{}, "q")
	if m.result != Exit || !m.done {
		t.Errorf("q: result=%v done=%v", m.result, m.done)
	}
}

func TestCursorClamps(t *testing.T) {
	m := send(model{}, "up", "up")
	if m.cursor != 0 {
		t.Errorf("cursor = %d, want 0", m.cursor)
	}
	m = send(m, "down", "down", "down", "down")
	if m.cursor != len(items)-1 {
		t.Errorf("cursor = %d, want %d", m.cursor, len(items)-1)
	}
}
