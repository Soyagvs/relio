// Package menu is the interactive main menu shown when `go-release` runs with no
// subcommand in a TTY. It is a thin selector; each action is carried out by the
// caller.
package menu

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/soyagvs/go-release/internal/ui"
)

// Action is the choice the user made in the menu.
type Action int

const (
	// None means the menu was dismissed without a choice (e.g. ctrl+c).
	None Action = iota
	CreateRelease
	ViewReleases
	GitHubAuth
	Help
	Exit
)

type item struct {
	label  string
	desc   string
	action Action
}

var items = []item{
	{"Create a release", "Version, changelog, and tag from commits since the last tag", CreateRelease},
	{"Releases", "List every version, read its notes, or delete one", ViewReleases},
	{"GitHub auth", "Log in with your own GitHub account (coming in v0.2.0)", GitHubAuth},
	{"Help", "Every command and flag, with a one-line description", Help},
	{"Exit", "Leave Go Release", Exit},
}

type model struct {
	cursor  int
	version string
	result  Action
	done    bool
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch key.String() {
	case "ctrl+c":
		m.result = None
		m.done = true
		return m, tea.Quit
	case "q", "esc":
		m.result = Exit
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
	return m, nil
}

func (m model) View() string {
	if m.done {
		// Leave a one-line trace in the scrollback instead of wiping the frame.
		return ui.Dim.Render("⬢ Go Release — menu closed") + "\n"
	}
	var b strings.Builder
	b.WriteString(ui.BigBanner(m.version) + "\n\n")

	for i, it := range items {
		cursor := "  "
		label := it.label
		desc := ui.Dim.Render(it.desc)
		if i == m.cursor {
			cursor = ui.Key.Render("▸ ")
			label = ui.Key.Render(label)
		}
		b.WriteString(cursor + label + "\n")
		b.WriteString("    " + desc + "\n\n")
	}

	b.WriteString(ui.Dim.Render("↑/↓ move · enter select · q quit"))
	return b.String()
}

// Run shows the menu once and returns the chosen Action.
func Run(version string) (Action, error) {
	final, err := tea.NewProgram(model{version: version}).Run()
	if err != nil {
		return None, err
	}
	return final.(model).result, nil
}
