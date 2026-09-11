// Package menu is the interactive main menu shown when `relio` runs with no
// subcommand in a TTY. It is a thin selector; each action is carried out by the
// caller.
package menu

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/soyagvs/relio/internal/i18n"
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
	Settings
	Guide
	Help
	Exit
)

type item struct {
	labelID i18n.MessageID
	descID  i18n.MessageID
	action  Action
	group   int // a faint rule is drawn wherever this changes between rows
}

// Label returns the rendered label for it, looked up at render time so it
// always reflects the active language (SetLanguage may run after items is
// initialized).
func (it item) Label() string { return i18n.T(it.labelID) }

// Desc mirrors Label for the item's description column.
func (it item) Desc() string { return i18n.T(it.descID) }

var items = []item{
	{i18n.MenuReleaseLabel, i18n.MenuReleaseDesc, Release, 1},
	{i18n.MenuStatusLabel, i18n.MenuStatusDesc, Status, 1},
	{i18n.MenuCheckLabel, i18n.MenuCheckDesc, Check, 1},
	{i18n.MenuViewReleasesLabel, i18n.MenuViewReleasesDesc, ViewReleases, 2},
	{i18n.MenuReleaseTextLabel, i18n.MenuReleaseTextDesc, ReleaseText, 2},
	{i18n.MenuReleaseImageLabel, i18n.MenuReleaseImageDesc, ReleaseImage, 2},
	{i18n.MenuAuthLabel, i18n.MenuAuthDesc, Auth, 3},
	{i18n.MenuSetupLabel, i18n.MenuSetupDesc, Setup, 3},
	{i18n.MenuSettingsLabel, i18n.MenuSettingsDesc, Settings, 3},
	{i18n.MenuGuideLabel, i18n.MenuGuideDesc, Guide, 3},
	{i18n.MenuHelpLabel, i18n.MenuHelpDesc, Help, 3},
	{i18n.MenuExitLabel, i18n.MenuExitDesc, Exit, 4},
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
	sepDim   = lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
	headline = lipgloss.NewStyle().Foreground(ui.Purple).Bold(true)
)

type model struct {
	cursor    int
	width     int // terminal width from tea.WindowSizeMsg; 0 until the first resize
	result    Action
	done      bool
	banner    bool
	version   string
	available string
	frames    []float64
	frame     int
}

type introTickMsg struct{}

func newModel(version, available string, animate bool) model {
	m := model{banner: true, version: version, available: available}
	if ui.BannerAnimationAllowed(animate) {
		m.frames = ui.BannerIntroFrames()
	}
	return m
}

func (m model) Init() tea.Cmd {
	if len(m.frames) > 1 {
		return introTick()
	}
	return nil
}

func introTick() tea.Cmd {
	return tea.Tick(ui.BannerIntroFrameDelay(), func(time.Time) tea.Msg { return introTickMsg{} })
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if _, ok := msg.(introTickMsg); ok {
		if m.done || len(m.frames) == 0 || m.frame >= len(m.frames)-1 {
			return m, nil
		}
		m.frame++
		if m.frame < len(m.frames)-1 {
			return m, introTick()
		}
		return m, nil
	}

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
		body := fmt.Sprintf("  %d  %s  %s", i+1, padLabel(it.Label()), it.Desc())
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

func (m model) View() string {
	if m.done {
		// The chosen action prints its own output next; stay quiet on exit.
		return ""
	}
	rowW := m.rowWidth()

	var b strings.Builder
	if m.banner {
		if len(m.frames) > 0 {
			b.WriteString(ui.BannerFrame(m.version, m.available, m.frames[m.frame]))
		} else {
			b.WriteString(ui.BigBanner(m.version, m.available))
		}
		b.WriteString("\n")
	}
	b.WriteString("  " + headline.Render(ui.Truncate(i18n.T(i18n.MenuHeadline, strings.ToUpper(ui.AppName)), max(0, rowW-2))) + "\n\n")

	for i, it := range items {
		// A faint rule wherever the group changes, so the menu reads in bands.
		if i > 0 && it.group != items[i-1].group {
			b.WriteString("  " + sepDim.Render(strings.Repeat("─", max(0, rowW-2))) + "\n")
		}

		num := fmt.Sprintf("%d", i+1)
		marker := "  "
		if i == m.cursor {
			marker = "▸ "
		}
		label := padLabel(it.Label())

		// Fixed-width prefix, then the description truncated so the whole body
		// fits in exactly rowW visible columns and nothing ever wraps.
		prefix := marker + num + "  " + label + "  "
		desc := ui.Truncate(it.Desc(), max(0, rowW-utf8.RuneCountInString(prefix)))

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

	hint := i18n.T(i18n.MenuHint)
	b.WriteString("\n  " + ui.Dim.Render(ui.Truncate(hint, max(0, rowW-2))))
	return b.String()
}

// Run shows the menu once and returns the chosen Action. The banner is rendered
// inside the Bubble Tea model so its eye can animate without blocking the menu.
func Run(version, available string, animate bool) (Action, error) {
	final, err := tea.NewProgram(newModel(version, available, animate)).Run()
	if err != nil {
		return None, err
	}
	return final.(model).result, nil
}
