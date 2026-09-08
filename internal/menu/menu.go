// Package menu is the interactive main menu shown when `relio` runs with no
// subcommand in a TTY. It is a thin selector; each action is carried out by the
// caller.
package menu

import (
	"fmt"
	"strings"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/soyagvs/relio/internal/ui"
)

// Action is the choice the user made in the menu.
type Action int

const (
	// None means the menu was dismissed without a choice (e.g. ctrl+c).
	None Action = iota
	Release
	Status
	Check
	ViewReleases
	ReleaseText
	ReleaseImage
	Auth
	Setup
	Guide
	Help
	Exit
)

type item struct {
	label  string
	desc   string
	action Action
}

var items = []item{
	{"Release", "Create a release — final or rc, and optionally push + publish", Release},
	{"Status", "What's unreleased and the version it suggests", Status},
	{"Check", "Which commits since the last tag are Conventional Commits", Check},
	{"Releases", "Browse versions, read notes, delete one", ViewReleases},
	{"Announcement", "Copy-paste release text — pick a format", ReleaseText},
	{"Release image", "Save or share a PNG release card", ReleaseImage},
	{"Auth", "GitHub connection — status and how to link", Auth},
	{"Setup", "Create or inspect .release.yaml", Setup},
	{"Guide", "Step-by-step walkthrough of the whole flow", Guide},
	{"Help", "Every command and flag", Help},
	{"Exit", "Leave Relio", Exit},
}

// digitRows is how many leading items answer to a 1–9 keypress; the rest
// (Help, Exit) are reachable with the arrow keys only.
const digitRows = 9

// Menu chrome: a fixed label column so the descriptions line up, a minimum row
// width, and a faint full-width bar behind the selected row.
const (
	labelCol    = 17
	minRowWidth = 40
)

var (
	selBar   = lipgloss.NewStyle().Background(lipgloss.Color("236"))
	numDim   = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	headline = lipgloss.NewStyle().Foreground(ui.Purple).Bold(true)
)

type model struct {
	cursor int
	width  int // terminal width from tea.WindowSizeMsg; 0 until the first resize
	result Action
	done   bool
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if ws, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = ws.Width
		return m, nil
	}

	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	s := key.String()
	switch s {
	case "ctrl+c":
		m.result = None
		m.done = true
		return m, tea.Quit
	case "q", "esc":
		m.result = Exit
		m.done = true
		return m, tea.Quit
	case "?":
		m.result = Help
		m.done = true
		return m, tea.Quit
	case "g":
		m.result = Guide
		m.done = true
		return m, tea.Quit
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(items)-1 {
			m.cursor++
		}
	case "enter", " ":
		m.result = items[m.cursor].action
		m.done = true
		return m, tea.Quit
	}

	// A digit jumps straight to that item and selects it. Only the first
	// digitRows items answer to a digit.
	if len(s) == 1 && s[0] >= '1' && s[0] <= '9' {
		if n := int(s[0] - '1'); n < digitRows && n < len(items) {
			m.cursor = n
			m.result = items[n].action
			m.done = true
			return m, tea.Quit
		}
	}
	return m, nil
}

// rowWidth is the visible column count every row (selected or not) is rendered
// to. It is the widest natural row, shrunk to fit the terminal (with one spare
// column so the terminal never soft-wraps) and never below minRowWidth unless
// the terminal itself is narrower.
func (m model) rowWidth() int {
	natural := 0
	for i, it := range items {
		// Plain body: "  " marker + num + "  " + padded label + "  " + desc.
		body := fmt.Sprintf("  %d  %s  %s", i+1, padLabel(it.label), it.desc)
		if w := utf8.RuneCountInString(body); w > natural {
			natural = w
		}
	}

	rowW := natural
	if rowW < minRowWidth {
		rowW = minRowWidth
	}
	if m.width > 0 && m.width-1 < rowW {
		rowW = m.width - 1
	}
	return rowW
}

func padLabel(label string) string {
	return label + strings.Repeat(" ", max(0, labelCol-utf8.RuneCountInString(label)))
}

// truncate shortens s to at most maxRunes visible runes, replacing the tail with
// "…" when it has to cut.
func truncate(s string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}
	if utf8.RuneCountInString(s) <= maxRunes {
		return s
	}
	r := []rune(s)
	if maxRunes == 1 {
		return "…"
	}
	return string(r[:maxRunes-1]) + "…"
}

func (m model) View() string {
	if m.done {
		// The chosen action prints its own output next; stay quiet on exit.
		return ""
	}
	rowW := m.rowWidth()

	var b strings.Builder
	b.WriteString("  " + headline.Render(truncate(strings.ToUpper(ui.AppName)+" menu", max(0, rowW-2))) + "\n\n")

	for i, it := range items {
		num := fmt.Sprintf("%d", i+1)
		marker := "  "
		if i == m.cursor {
			marker = "▸ "
		}
		label := padLabel(it.label)

		// Fixed-width prefix, then the description truncated so the whole body
		// fits in exactly rowW visible columns and nothing ever wraps.
		prefix := marker + num + "  " + label + "  "
		desc := truncate(it.desc, max(0, rowW-utf8.RuneCountInString(prefix)))

		// Colorize the already-fitted pieces; visible widths are unchanged.
		numOut, labelOut := numDim.Render(num), label
		if i == m.cursor {
			numOut, labelOut = ui.Key.Render(num), ui.Key.Render(label)
		}
		body := marker + numOut + "  " + labelOut + "  " + ui.Dim.Render(desc)

		if i == m.cursor {
			// Width(rowW) now equals the body width, so it pads (fills the bar)
			// without ever wrapping to a second line.
			b.WriteString(selBar.Width(rowW).Render(body) + "\n")
			continue
		}

		// Pad non-selected rows to rowW too, so the trailing newline always
		// lands at the same column and the inline renderer's math stays stable.
		visible := utf8.RuneCountInString(prefix) + utf8.RuneCountInString(desc)
		if pad := rowW - visible; pad > 0 {
			body += strings.Repeat(" ", pad)
		}
		b.WriteString(body + "\n")
	}

	hint := "↑/↓ move · 1–9 jump · ? help · g guide · enter select · q quit"
	b.WriteString("\n  " + ui.Dim.Render(truncate(hint, max(0, rowW-2))))
	return b.String()
}

// Run shows the menu once and returns the chosen Action. The big entry banner is
// printed by the caller before Run, so it lands in scrollback; View() itself
// stays short enough for a small terminal.
func Run() (Action, error) {
	final, err := tea.NewProgram(model{}).Run()
	if err != nil {
		return None, err
	}
	return final.(model).result, nil
}
