// Package menu is the interactive main menu shown when `relio` runs with no
// subcommand in a TTY. It is a thin selector; each action is carried out by the
// caller.
package menu

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/soyagvs/relio/internal/ui"
)

// Action is the choice the user made in the menu.
type Action int

const (
	// None means the menu was dismissed without a choice (e.g. ctrl+c).
	None Action = iota
	Status
	CreateRelease
	ViewReleases
	ReleaseText
	ReleaseImage
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
	{"Status", "What's unreleased since the last tag and the version it suggests", Status},
	{"Create a release", "Version, changelog, and tag from commits since the last tag", CreateRelease},
	{"Releases", "List every version, read its notes, or delete one", ViewReleases},
	{"Release text", "Copy-paste announcement for social posts — pick a format", ReleaseText},
	{"Release image", "Save a shareable PNG of a release — pick a shape", ReleaseImage},
	{"GitHub auth", "Log in with your own GitHub account (coming in v0.2.0)", GitHubAuth},
	{"Help", "Every command and flag, with a one-line description", Help},
	{"Exit", "Leave Relio", Exit},
}

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
		// The chosen action prints its own output next; stay quiet on exit.
		return ""
	}
	var b strings.Builder
	b.WriteString(ui.BigBanner(m.version, m.update) + "\n\n")

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

// Run shows the menu once and returns the chosen Action. update, when non-empty,
// is a bare newer version ("1.2.3") shown next to the current one in the banner.
func Run(version, update string) (Action, error) {
	final, err := tea.NewProgram(model{version: version, update: update}).Run()
	if err != nil {
		return None, err
	}
	return final.(model).result, nil
}
