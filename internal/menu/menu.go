// Package menu is the interactive main menu shown when `relio` runs with no
// subcommand in a TTY. It is a thin selector; each action is carried out by the
// caller.
package menu

import (
	"fmt"
	"strings"

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
	Stats
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
	{"Stats", "Public download and star numbers", Stats},
	{"Auth", "GitHub connection — status and how to link", Auth},
	{"Setup", "Create or inspect .release.yaml", Setup},
	{"Guide", "Step-by-step walkthrough of the whole flow", Guide},
	{"Help", "Every command and flag", Help},
	{"Exit", "Leave Relio", Exit},
}

// digitRows is how many leading items answer to a 1–9 keypress; the rest
// (Guide, Help, Exit) are reachable with the arrow keys only.
const digitRows = 9

// Menu chrome: a fixed label column so the descriptions line up, and a faint
// full-width bar behind the selected row.
const (
	labelCol = 17
	rowWidth = 64
)

var (
	selBar   = lipgloss.NewStyle().Background(lipgloss.Color("236")).Width(rowWidth)
	numDim   = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	headline = lipgloss.NewStyle().Foreground(ui.Purple).Bold(true)
)

type model struct {
	cursor  int
	version string
	update  string // newer version available as a bare "1.2.3", or "" when current
	result  Action
	done    bool
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

// banner is the big entry banner. It is printed once by Run before the program
// starts (so it lands in scrollback), never inside View, which must stay short
// enough to fit a small terminal without the top scrolling off.
func (m model) banner() string { return ui.BigBanner(m.version, m.update) }

func (m model) View() string {
	if m.done {
		// The chosen action prints its own output next; stay quiet on exit.
		return ""
	}
	var b strings.Builder
	b.WriteString("  " + headline.Render(strings.ToUpper(ui.AppName)+" menu") + "\n\n")

	for i, it := range items {
		num := fmt.Sprintf("%d", i+1)
		label := it.label + strings.Repeat(" ", max(0, labelCol-len(it.label)))

		if i == m.cursor {
			inner := fmt.Sprintf(" %s  %s  %s",
				ui.Key.Render(num), ui.Key.Render(label), ui.Dim.Render(it.desc))
			b.WriteString(ui.Key.Render("▸") + selBar.Render(inner) + "\n")
			continue
		}
		b.WriteString(fmt.Sprintf("  %s  %s  %s\n",
			numDim.Render(num), label, ui.Dim.Render(it.desc)))
	}

	b.WriteString("\n  " + ui.Dim.Render("↑/↓ move · 1–9 jump · ? help · g guide · enter select · q quit"))
	return b.String()
}

// Run shows the menu once and returns the chosen Action. update, when non-empty,
// is a bare newer version ("1.2.3") shown next to the current one in the banner.
func Run(version, update string) (Action, error) {
	m := model{version: version, update: update}
	fmt.Print(m.banner() + "\n")
	final, err := tea.NewProgram(m).Run()
	if err != nil {
		return None, err
	}
	return final.(model).result, nil
}
