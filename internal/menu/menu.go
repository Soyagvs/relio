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
	Status
	Check
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
	{"Status", "Unreleased commits and the next version", Status},
	{"Check", "Lint the commits since the last tag — which are Conventional Commits, and the bump", Check},
	{"Create a release", "Version, changelog, and tag", CreateRelease},
	{"Releases", "Browse, read notes, or delete a version", ViewReleases},
	{"Release text", "Announcement text — pick a format", ReleaseText},
	{"Release image", "Shareable PNG — pick a shape", ReleaseImage},
	{"GitHub auth", "Token-based; check it with `relio auth status`", GitHubAuth},
	{"Help", "Every command and flag", Help},
	{"Exit", "Leave Relio", Exit},
}

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

	// A digit jumps straight to that item and selects it.
	if len(s) == 1 && s[0] >= '1' && s[0] <= '9' {
		if n := int(s[0] - '1'); n < len(items) {
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

	b.WriteString("\n  " + ui.Dim.Render("↑/↓ move · 1–9 jump · enter select · q quit"))
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
