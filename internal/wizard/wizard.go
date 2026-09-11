// Package wizard is the interactive confirmation step for `release`, built with
// Bubble Tea. It is used only when stdin/stdout is a TTY; non-interactive runs
// bypass it entirely.
package wizard

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/soyagvs/relio/internal/i18n"
	"github.com/soyagvs/relio/internal/release"
	"github.com/soyagvs/relio/internal/semver"
	"github.com/soyagvs/relio/internal/ui"
)

// Result is the wizard's outcome.
type Result struct {
	Confirmed bool
	// Bump is the (possibly overridden) bump the user settled on.
	Bump semver.Bump
}

type choice struct {
	label string
	// bump is the override this choice applies before confirming; semver.None
	// means "keep whatever is current".
	bump    semver.Bump
	confirm bool
	cancel  bool
}

type model struct {
	plan     release.Plan
	current  semver.Bump // currently selected bump (starts at plan.Bump)
	cursor   int
	choices  []choice
	result   Result
	quitting bool
}

func newModel(p release.Plan) model {
	return model{
		plan:    p,
		current: p.Bump,
		choices: []choice{
			{label: i18n.T(i18n.WizardChoiceConfirm), confirm: true},
			{label: i18n.T(i18n.WizardChoicePatch), bump: semver.Patch},
			{label: i18n.T(i18n.WizardChoiceMinor), bump: semver.Minor},
			{label: i18n.T(i18n.WizardChoiceMajor), bump: semver.Major},
			{label: i18n.T(i18n.WizardChoiceCancel), cancel: true},
		},
	}
}

func (m model) Init() tea.Cmd { return nil }

// nextVersion is the version shown in the wizard header. While the user keeps
// the plan's own bump it shows plan.Next verbatim, so a pre-release or a
// finalising run reads correctly; once they pick a different bump it previews
// that bump's core off the current version.
func (m model) nextVersion() semver.Version {
	if m.current == m.plan.Bump {
		return m.plan.Next
	}
	return m.plan.Current.Next(m.current)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch key.String() {
	case "ctrl+c", "q", "esc":
		m.result = Result{Confirmed: false}
		m.quitting = true
		return m, tea.Quit
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.choices)-1 {
			m.cursor++
		}
	case "y", "Y":
		m.result = Result{Confirmed: true, Bump: m.current}
		m.quitting = true
		return m, tea.Quit
	case "enter", " ":
		c := m.choices[m.cursor]
		switch {
		case c.cancel:
			m.result = Result{Confirmed: false}
			m.quitting = true
			return m, tea.Quit
		case c.confirm:
			m.result = Result{Confirmed: true, Bump: m.current}
			m.quitting = true
			return m, tea.Quit
		default:
			m.current = c.bump
			m.cursor = 0 // jump back to "Create this release"
		}
	}
	return m, nil
}

func (m model) View() string {
	if m.quitting {
		// The plan preview was already printed to the scrollback by the caller;
		// leave only a short trace of the decision here.
		if m.result.Confirmed {
			return ui.Dim.Render(fmt.Sprintf(i18n.T(i18n.WizardConfirmed), m.nextVersion().String())) + "\n"
		}
		return ui.Dim.Render(i18n.T(i18n.WizardCancelled)) + "\n"
	}

	var b strings.Builder
	b.WriteString(ui.Key.Render(fmt.Sprintf(i18n.T(i18n.WizardHeader), m.nextVersion().String())) +
		ui.Dim.Render("   "+fmt.Sprintf(i18n.T(i18n.WizardBumpFrom), m.current, m.plan.Current)) + "\n\n")

	for i, c := range m.choices {
		cursor := "  "
		line := c.label
		if c.confirm {
			line = fmt.Sprintf(line, m.nextVersion().String())
		}
		if i == m.cursor {
			cursor = ui.Key.Render("▸ ")
			line = ui.Key.Render(line)
		}
		b.WriteString(cursor + line + "\n")
	}

	b.WriteString("\n" + ui.Dim.Render(i18n.T(i18n.WizardHint)))
	return b.String()
}

// Run displays the wizard and blocks until the user confirms or cancels.
func Run(p release.Plan) (Result, error) {
	m := newModel(p)
	prog := tea.NewProgram(m)
	final, err := prog.Run()
	if err != nil {
		return Result{}, err
	}
	return final.(model).result, nil
}
