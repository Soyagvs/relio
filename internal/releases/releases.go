// Package releases is the interactive release browser reachable from the main
// menu: list every version, read its notes, and delete one (tag + changelog
// section).
package releases

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/soyagvs/go-release/internal/changelog"
	"github.com/soyagvs/go-release/internal/config"
	"github.com/soyagvs/go-release/internal/gitrepo"
	"github.com/soyagvs/go-release/internal/ui"
)

// repoPort is the slice of *gitrepo.Repo this package needs; it keeps the model
// testable with a fake.
type repoPort interface {
	Tags() ([]gitrepo.TagInfo, error)
	TagMessage(name string) (string, error)
	DeleteTag(name string) error
}

type mode int

const (
	browse mode = iota
	confirmDelete
)

type model struct {
	repo          repoPort
	changelogPath string
	changelog     string

	tags   []gitrepo.TagInfo
	cursor int
	mode   mode
	status string
	err    string
	quit   bool
}

func newModel(repo repoPort, changelogPath string) model {
	m := model{repo: repo, changelogPath: changelogPath}
	m.reload()
	return m
}

func (m *model) reload() {
	tags, err := m.repo.Tags()
	if err != nil {
		m.err = err.Error()
		return
	}
	m.tags = tags
	if m.cursor >= len(tags) {
		m.cursor = max(0, len(tags)-1)
	}
	data, _ := os.ReadFile(m.changelogPath)
	m.changelog = string(data)
}

func (m model) Init() tea.Cmd { return nil }

func (m model) selected() (gitrepo.TagInfo, bool) {
	if len(m.tags) == 0 || m.cursor < 0 || m.cursor >= len(m.tags) {
		return gitrepo.TagInfo{}, false
	}
	return m.tags[m.cursor], true
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	if m.mode == confirmDelete {
		switch key.String() {
		case "y", "Y":
			m = m.doDelete()
		case "n", "N", "esc", "q":
			m.mode = browse
			m.status = "Delete cancelled."
		}
		return m, nil
	}

	switch key.String() {
	case "ctrl+c", "q", "esc":
		m.quit = true
		return m, tea.Quit
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
			m.status = ""
		}
	case "down", "j":
		if m.cursor < len(m.tags)-1 {
			m.cursor++
			m.status = ""
		}
	case "d", "x":
		if _, ok := m.selected(); ok {
			m.mode = confirmDelete
			m.status = ""
		}
	}
	return m, nil
}

func (m model) doDelete() model {
	tag, ok := m.selected()
	m.mode = browse
	if !ok {
		return m
	}
	if err := m.repo.DeleteTag(tag.Name); err != nil {
		m.err = err.Error()
		return m
	}

	removed := "tag"
	if m.changelog != "" && changelog.ExtractSection(m.changelog, tag.Name) != "" {
		updated := changelog.RemoveSection(m.changelog, tag.Name)
		if err := os.WriteFile(m.changelogPath, []byte(updated), 0o644); err != nil {
			m.err = fmt.Sprintf("tag deleted, but changelog: %v", err)
			m.reload()
			return m
		}
		removed = "tag + changelog section"
	}

	m.reload()
	m.status = fmt.Sprintf("Deleted %s (%s).", tag.Name, removed)
	return m
}

// notesFor returns the text shown in the detail pane for a tag.
func (m model) notesFor(tag gitrepo.TagInfo) string {
	if s := changelog.ExtractSection(m.changelog, tag.Name); s != "" {
		return s
	}
	if msg, _ := m.repo.TagMessage(tag.Name); strings.TrimSpace(msg) != "" {
		return strings.TrimSpace(msg)
	}
	return ui.Dim.Render("(no notes for this version)")
}

var (
	paneStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder(), false, false, false, true).
			BorderForeground(lipgloss.Color("245")).
			PaddingLeft(2).
			MarginTop(1)
	rowSel = lipgloss.NewStyle().Foreground(ui.Orange).Bold(true)
)

func (m model) View() string {
	if m.quit {
		return m.staticView()
	}

	var b strings.Builder
	b.WriteString(ui.Title.Render("⬢ Releases") + "\n\n")

	if m.err != "" {
		b.WriteString(ui.Warn.Render("✗ "+m.err) + "\n\n")
	}
	if len(m.tags) == 0 {
		b.WriteString(ui.Dim.Render("No releases yet. Create one from the menu.") + "\n\n")
		b.WriteString(ui.Dim.Render("q back"))
		return b.String()
	}

	for i, t := range m.tags {
		line := fmt.Sprintf("%-12s  %s  %s", t.Name, t.Date, ui.Dim.Render(t.Subject))
		if i == m.cursor {
			b.WriteString(ui.Key.Render("▸ ") + rowSel.Render(fmt.Sprintf("%-12s", t.Name)) +
				ui.Dim.Render("  "+t.Date+"  ") + t.Subject + "\n")
		} else {
			b.WriteString("  " + line + "\n")
		}
	}

	if tag, ok := m.selected(); ok {
		b.WriteString(paneStyle.Render(m.notesFor(tag)))
		b.WriteString("\n")
	}

	if m.mode == confirmDelete {
		tag, _ := m.selected()
		b.WriteString("\n" + ui.Warn.Render(fmt.Sprintf("Delete %s? This removes the git tag and its changelog section.  [y/N]", tag.Name)))
	} else {
		if m.status != "" {
			b.WriteString("\n" + ui.Ok.Render("✓ ") + m.status)
		}
		b.WriteString("\n\n" + ui.Dim.Render("↑/↓ move · d delete · q back"))
	}
	return b.String()
}

// staticView is what remains in the scrollback after quitting.
func (m model) staticView() string {
	var b strings.Builder
	b.WriteString(ui.Title.Render("⬢ Releases") + "\n")
	if len(m.tags) == 0 {
		b.WriteString(ui.Dim.Render("  (none)") + "\n")
	}
	for _, t := range m.tags {
		b.WriteString(fmt.Sprintf("  %-12s  %s  %s\n", t.Name, t.Date, ui.Dim.Render(t.Subject)))
	}
	if m.status != "" {
		b.WriteString(ui.Dim.Render("  "+m.status) + "\n")
	}
	return b.String()
}

// Run opens the browser against repo and blocks until the user goes back.
func Run(repo *gitrepo.Repo, cfg config.Config) error {
	path := cfg.Release.ChangelogFile
	if path == "" {
		path = "CHANGELOG.md"
	}
	m := newModel(repo, repo.Root()+string(os.PathSeparator)+path)
	_, err := tea.NewProgram(m).Run()
	return err
}
