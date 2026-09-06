// Package pick is a small reusable single-select list built with Bubble Tea,
// used for sub-choices reached from the main menu (e.g. the post format).
package pick

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/soyagvs/go-release/internal/ui"
)

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
	quit   bool
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	k, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch k.String() {
	case "ctrl+c", "q", "esc":
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
// backed out with q/esc.
func Run(title string, items []Item) (value string, chosen bool, err error) {
	final, e := tea.NewProgram(model{title: title, items: items}).Run()
	if e != nil {
		return "", false, e
	}
	m := final.(model)
	if !m.chosen {
		return "", false, nil
	}
	return m.items[m.cursor].Value, true, nil
}
