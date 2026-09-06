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

func TestFirstItemIsStatus(t *testing.T) {
	m := send(model{}, "enter")
	if m.result != Status {
		t.Errorf("result = %v, want Status (the first item)", m.result)
	}
}

// selectAction drives the menu down to the item with the given action and picks it.
func selectAction(a Action) model {
	m := model{}
	for _, it := range items {
		if it.action == a {
			break
		}
		m = send(m, "down")
	}
	return send(m, "enter")
}

func TestSelectEveryAction(t *testing.T) {
	for _, a := range []Action{Status, CreateRelease, ViewReleases, ReleaseText, ReleaseImage, GitHubAuth, Help, Exit} {
		if got := selectAction(a).result; got != a {
			t.Errorf("selecting %v gave %v", a, got)
		}
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
	for range items {
		m = send(m, "down")
	}
	if m.cursor != len(items)-1 {
		t.Errorf("cursor = %d, want %d", m.cursor, len(items)-1)
	}
}
