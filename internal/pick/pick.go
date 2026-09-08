// Package pick is a small reusable single-select list built with Bubble Tea,
// used for sub-choices reached from the main menu (e.g. the post format).
package pick

import (
	"errors"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/soyagvs/relio/internal/ui"
)

// ErrQuit is returned by Run when the user hard-quits with ctrl+c, as opposed to
// backing out of the list with q/esc (which returns chosen=false, err=nil).
var ErrQuit = errors.New("pick: quit")

// Item is one selectable option. Value is what Run returns.
type Item struct {
	Label string
	Desc  string
	Value string
}

type model struct {
	title  string
	items  []Item
	cursor int
	chosen bool
	quit   bool // backed out with q/esc
	killed bool // hard-quit with ctrl+c
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	k, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch k.String() {
	case "ctrl+c":
		m.quit = true
		m.killed = true
		return m, tea.Quit
	case "q", "esc":
		m.quit = true
		return m, tea.Quit
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.items)-1 {
			m.cursor++
		}
	case "enter", " ":
		m.chosen = true
		return m, tea.Quit
	}
	return m, nil
}

func (m model) View() string {
	if m.chosen {
		return ui.Dim.Render("→ "+m.items[m.cursor].Label) + "\n"
	}
	if m.quit {
		return ui.Dim.Render("→ cancelled") + "\n"
	}

	var b strings.Builder
	b.WriteString(ui.Title.Render(m.title) + "\n\n")
	for i, it := range m.items {
		cur, label := "  ", it.Label
		if i == m.cursor {
			cur, label = ui.Key.Render("▸ "), ui.Key.Render(it.Label)
		}
		b.WriteString(cur + label + "\n")
		if it.Desc != "" {
			b.WriteString("    " + ui.Dim.Render(it.Desc) + "\n")
		}
	}
	b.WriteString("\n" + ui.Dim.Render("↑/↓ move · enter select · q cancel"))
	return b.String()
}

// Run shows the list and returns the chosen Value. chosen is false when the user
// backed out with q/esc (err is nil then); a ctrl+c hard-quit returns ErrQuit.
func Run(title string, items []Item) (value string, chosen bool, err error) {
	final, e := tea.NewProgram(model{title: title, items: items}).Run()
	if e != nil {
		return "", false, e
	}
	m := final.(model)
	if m.killed {
		return "", false, ErrQuit
	}
	if !m.chosen {
		return "", false, nil
	}
	return m.items[m.cursor].Value, true, nil
}
