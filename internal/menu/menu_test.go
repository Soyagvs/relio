package menu

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

func TestFirstItemIsRelease(t *testing.T) {
	m := send(model{}, "enter")
	if m.result != Release {
		t.Errorf("result = %v, want Release (the first item)", m.result)
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
	for _, a := range []Action{
		Release, Status, Check, ViewReleases, ReleaseText, ReleaseImage,
		Auth, Setup, Guide, Help, Exit,
	} {
		if got := selectAction(a).result; got != a {
			t.Errorf("selecting %v gave %v", a, got)
		}
	}
}

func TestHelpAndGuideKeys(t *testing.T) {
	if m := send(model{}, "?"); m.result != Help || !m.done {
		t.Errorf("? key: result=%v done=%v, want Help", m.result, m.done)
	}
	if m := send(model{}, "g"); m.result != Guide || !m.done {
		t.Errorf("g key: result=%v done=%v, want Guide", m.result, m.done)
	}
}

// Rows 1–9 answer to a digit; rows 10–11 (Help, Exit) are arrow-only.
func TestDigitJumpStopsAtRowNine(t *testing.T) {
	if m := send(model{}, "9"); m.result != Guide {
		t.Errorf(`"9" => %v, want Guide (row 9)`, m.result)
	}
}

func TestCheckItemPresent(t *testing.T) {
	found := false
	for _, it := range items {
		if it.action == Check {
			found = true
			if it.label != "Check" {
				t.Errorf("Check label = %q", it.label)
			}
			if it.desc == "" {
				t.Error("Check needs a description")
			}
		}
	}
	if !found {
		t.Error("no menu item wired to Check")
	}
}

func TestQuitKeyIsExit(t *testing.T) {
	m := send(model{}, "q")
	if m.result != Exit || !m.done {
		t.Errorf("q: result=%v done=%v", m.result, m.done)
	}
}

// The banner is printed once by the caller before the program starts; the live
// View() must stay short so it fits a small terminal without the top scrolling
// off, and must not embed the big wordmark itself.
func TestViewOmitsBanner(t *testing.T) {
	v := model{width: 80}.View()
	if strings.Contains(v, "█") {
		t.Error("View() must not embed the big wordmark banner")
	}
	if got := strings.Count(v, "\n") + 1; got > 20 {
		t.Errorf("View() is %d lines, too tall for a small terminal", got)
	}
}

// Moving the cursor must never change how many lines View() renders, nor let any
// line exceed the terminal width — that line-count drift is what made the menu
// "deform" as the selection moved.
func TestViewLineCountStableAcrossCursor(t *testing.T) {
	for _, w := range []int{80, 50} {
		want := -1
		for i := range items {
			v := model{width: w, cursor: i}.View()
			if n := strings.Count(v, "\n"); want == -1 {
				want = n
			} else if n != want {
				t.Errorf("width %d: View() at cursor %d has %d newlines, want %d (cursor 0)", w, i, n, want)
			}
			for _, ln := range strings.Split(v, "\n") {
				if lipgloss.Width(ln) > w {
					t.Errorf("width %d: line at cursor %d exceeds terminal (%d cols): %q", w, i, lipgloss.Width(ln), ln)
				}
			}
		}
	}
}

// A faint rule is drawn at every group boundary; the count must match the
// number of transitions in the items list and not depend on the cursor.
func TestViewHasGroupSeparators(t *testing.T) {
	boundaries := 0
	for i := 1; i < len(items); i++ {
		if items[i].group != items[i-1].group {
			boundaries++
		}
	}
	if boundaries == 0 {
		t.Fatal("no group boundaries in items — the test is meaningless")
	}
	for _, cur := range []int{0, 5, len(items) - 1} {
		v := model{width: 80, cursor: cur}.View()
		if got := strings.Count(v, "─"); got == 0 {
			t.Errorf("cursor %d: View() has no separator rule", cur)
		}
		lines := 0
		for _, ln := range strings.Split(v, "\n") {
			if strings.Contains(ln, "─") {
				lines++
			}
		}
		if lines != boundaries {
			t.Errorf("cursor %d: %d separator lines, want %d", cur, lines, boundaries)
		}
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
