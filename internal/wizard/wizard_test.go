package wizard

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/soyagvs/go-release/internal/config"
	"github.com/soyagvs/go-release/internal/release"
	"github.com/soyagvs/go-release/internal/semver"
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
