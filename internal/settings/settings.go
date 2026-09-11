// Package settings is the interactive Settings screen: a Language section
// (single-select radio) and a Changelog-footer section (two independent
// checkboxes for the current project's .release.yaml), sharing one cursor.
// Every change persists immediately — there is no separate "save" action.
package settings

import (
	"errors"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/soyagvs/relio/internal/config"
	"github.com/soyagvs/relio/internal/i18n"
	"github.com/soyagvs/relio/internal/ui"
	"github.com/soyagvs/relio/internal/userconfig"
)

// ErrQuit is returned by Run when the user hard-quits with ctrl+c, rather
// than leaving the screen with q/esc (which returns nil — every change is
// already persisted, so there is nothing to confirm).
var ErrQuit = errors.New("settings: quit")

// kind is the shape of one row in the screen.
type kind int

const (
	// rowInfo is a non-selectable row: a section title or the disabled-footer
	// reason. It never takes the cursor.
	rowInfo kind = iota
	rowRadio
	rowCheck
)

// row is one line of the Settings screen. label/desc are closures re-evaluated
// on every View() so switching the active language repaints this very screen
// on the next frame with no model rebuild.
type row struct {
	kind    kind
	label   func() string
	desc    func() string
	value   string   // language id, for rowRadio
	field   []string // config.Field path, for rowCheck
	group   int      // a faint rule is drawn where this changes, like the menu
	enabled bool
}

// model is the Settings screen's Bubble Tea model.
type model struct {
	rows   []row
	cursor int
	width  int

	root string // repo root; "" when run outside a git repo
	lang string // active i18n language id

	cfg config.Config // only meaningful when footer is true

	err    string // last persistence failure, shown inline
	done   bool
	killed bool // hard-quit with ctrl+c
}

// newModel loads the current state (global language +, when inside a repo
// with a .release.yaml, the project config) and builds the row list.
func newModel(root string) model {
	m := model{root: root, lang: i18n.Current()}

	footer := false
	if root != "" {
		if cfg, err := config.Load(root); err == nil {
			m.cfg = cfg
			footer = true
		}
	}
	m.rows = buildRows(footer)
	m.cursor = m.firstEnabledIndex()
	return m
}

// buildRows constructs the static row list. footer controls whether the two
// checkbox rows are enabled, or whether a disabled-reason row is shown
// instead.
func buildRows(footer bool) []row {
	rows := []row{
		{kind: rowInfo, label: func() string { return i18n.T(i18n.SettingsLanguageSection) }, group: 1},
	}
	for _, lang := range i18n.Languages() {
		lang := lang
		rows = append(rows, row{
			kind:    rowRadio,
			label:   func() string { return lang.Name },
			value:   lang.ID,
			group:   1,
			enabled: true,
		})
	}

	rows = append(rows, row{kind: rowInfo, label: func() string { return i18n.T(i18n.SettingsFooterSection) }, group: 2})
	rows = append(rows,
		row{
			kind:    rowCheck,
			label:   func() string { return i18n.T(i18n.SettingsContributorsLabel) },
			desc:    func() string { return i18n.T(i18n.SettingsContributorsDesc) },
			field:   []string{"release", "contributors"},
			group:   2,
			enabled: footer,
		},
		row{
			kind:    rowCheck,
			label:   func() string { return i18n.T(i18n.SettingsCompareLinkLabel) },
			desc:    func() string { return i18n.T(i18n.SettingsCompareLinkDesc) },
			field:   []string{"release", "compare_link"},
			group:   2,
			enabled: footer,
		},
	)
	if !footer {
		rows = append(rows, row{kind: rowInfo, label: func() string { return i18n.T(i18n.SettingsFooterDisabledReason) }, group: 2})
	}
	return rows
}

func (m model) firstEnabledIndex() int {
	for i, r := range m.rows {
		if r.enabled {
			return i
		}
	}
	return 0
}

// checkValue reads the current in-memory value for a rowCheck's config
// field path.
func (m model) checkValue(field []string) bool {
	if len(field) == 2 && field[0] == "release" {
		switch field[1] {
		case "contributors":
			return m.cfg.Release.Contributors
		case "compare_link":
			return m.cfg.Release.CompareLink
		}
	}
	return false
}

// setCheckValue writes v into the in-memory config field addressed by field.
func (m *model) setCheckValue(field []string, v bool) {
	if len(field) == 2 && field[0] == "release" {
		switch field[1] {
		case "contributors":
			m.cfg.Release.Contributors = v
		case "compare_link":
			m.cfg.Release.CompareLink = v
		}
	}
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

	switch key.String() {
	case "ctrl+c":
		m.done, m.killed = true, true
		return m, tea.Quit
	case "q", "esc":
		m.done = true
		return m, tea.Quit
	case "up", "k":
		return m.moveCursor(-1), nil
	case "down", "j":
		return m.moveCursor(1), nil
	case "enter", " ":
		return m.apply(), nil
	}
	return m, nil
}

// moveCursor steps the cursor by delta, skipping disabled rows (section
// headers and, outside a repo, the footer checkboxes). It clamps at the
// first/last enabled row rather than wrapping.
func (m model) moveCursor(delta int) model {
	n := len(m.rows)
	i := m.cursor
	for step := 0; step < n; step++ {
		i += delta
		if i < 0 || i >= n {
			return m
		}
		if m.rows[i].enabled {
			m.cursor = i
			return m
		}
	}
	return m
}

// apply handles enter/space on the current row: select a language, or toggle
// a footer checkbox. Both persist immediately; a checkbox reverts its
// in-memory flip if the persistence call fails, so the screen never lies
// about disk state.
func (m model) apply() model {
	if m.cursor < 0 || m.cursor >= len(m.rows) {
		return m
	}
	r := m.rows[m.cursor]
	switch r.kind {
	case rowRadio:
		resolved, _ := i18n.SetLanguage(r.value)
		m.lang = resolved
		if err := userconfig.Save(userconfig.Config{Language: resolved}); err != nil {
			m.err = err.Error()
		} else {
			m.err = ""
		}
	case rowCheck:
		if !r.enabled {
			return m
		}
		before := m.checkValue(r.field)
		next := !before
		m.setCheckValue(r.field, next)
		if err := config.SetFields(m.root, config.Field{Path: r.field, Value: next}); err != nil {
			m.err = err.Error()
			m.setCheckValue(r.field, before)
		} else {
			m.err = ""
		}
	}
	return m
}

func (m model) View() string {
	if m.done {
		return ""
	}

	var b strings.Builder
	b.WriteString("  " + ui.Title.Render(i18n.T(i18n.SettingsTitle)) + "\n\n")

	for i, r := range m.rows {
		if i > 0 && r.group != m.rows[i-1].group {
			b.WriteString("\n")
		}

		marker := "  "
		if i == m.cursor {
			marker = "▸ "
		}

		switch r.kind {
		case rowInfo:
			b.WriteString("  " + ui.Dim.Render(r.label()) + "\n")

		case rowRadio:
			box := "[ ]"
			if r.value == m.lang {
				box = "[x]"
			}
			label := r.label()
			if i == m.cursor {
				box, label = ui.Key.Render(box), ui.Key.Render(label)
			}
			b.WriteString("  " + marker + box + " " + label + "\n")

		case rowCheck:
			box := "[ ]"
			if m.checkValue(r.field) {
				box = "[x]"
			}
			label := r.label()
			desc := ""
			if r.desc != nil {
				desc = "  " + ui.Dim.Render(r.desc())
			}
			switch {
			case !r.enabled:
				box, label = ui.Dim.Render(box), ui.Dim.Render(label)
			case i == m.cursor:
				box, label = ui.Key.Render(box), ui.Key.Render(label)
			}
			b.WriteString("  " + marker + box + " " + label + desc + "\n")
		}
	}

	if m.err != "" {
		b.WriteString("\n  " + ui.Warn.Render(m.err) + "\n")
	}

	b.WriteString("\n  " + ui.Dim.Render(ui.Truncate(i18n.T(i18n.SettingsHint), max(0, m.hintWidth()))))
	return b.String()
}

// hintWidth keeps the hint from wrapping on a narrow terminal; 0 (no known
// width yet) truncates to nothing extra since Truncate(s, 0) == "".
func (m model) hintWidth() int {
	if m.width <= 0 {
		return 1 << 20 // effectively unbounded until the first resize
	}
	return m.width - 2
}

// Run shows the Settings screen and blocks until the user leaves it. root is
// the current project's repo root, or "" when run outside a git repository —
// the Language section stays usable either way; the Changelog-footer section
// is disabled without a repo.
func Run(root string) error {
	final, err := tea.NewProgram(newModel(root)).Run()
	if err != nil {
		return err
	}
	if final.(model).killed {
		return ErrQuit
	}
	return nil
}
