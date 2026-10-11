// Package releases is the interactive release browser reachable from the main
// menu: list every version, read its notes, and delete one (tag + changelog
// section).
package releases

import (
	"errors"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/soyagvs/relio/internal/changelog"
	"github.com/soyagvs/relio/internal/config"
	"github.com/soyagvs/relio/internal/conventional"
	"github.com/soyagvs/relio/internal/gitrepo"
	"github.com/soyagvs/relio/internal/i18n"
	"github.com/soyagvs/relio/internal/ui"
)

// repoPort is the slice of *gitrepo.Repo this package needs; it keeps the model
// testable with a fake.
type repoPort interface {
	Tags() ([]gitrepo.TagInfo, error)
	TagMessage(name string) (string, error)
	DeleteTag(name string) error
	CommitsBetween(from, to string) ([]conventional.Raw, error)
}

// ErrQuit is returned by Run when the user hard-quits with ctrl+c, as opposed
// to leaving the browser with q/esc (which returns nil).
var ErrQuit = errors.New("releases: quit")

type mode int

const (
	browse mode = iota
	preview
	confirmDelete
)

type model struct {
	repo          repoPort
	project       string
	changelogPath string
	changelog     string

	tags           []gitrepo.TagInfo
	cursor         int
	mode           mode
	status         string
	err            string
	quit           bool
	killed         bool // hard-quit with ctrl+c
	picked         int  // retained for older static output paths; -1 = none
	selectable     bool
	confirmed      bool
	detailOffset   int
	previewBack    bool
	previewConfirm bool
}

const backLabel = "<- Back"
const backHomeLabel = "<- Back to home"
const releasesPageSize = 10

func newModel(repo repoPort, project, changelogPath string) model {
	m := model{repo: repo, project: project, changelogPath: changelogPath, picked: -1}
	m.reload()
	return m
}

func newSelectModel(repo repoPort, project, changelogPath string) model {
	m := newModel(repo, project, changelogPath)
	m.selectable = true
	return m
}

func (m *model) reload() {
	tags, err := m.repo.Tags()
	if err != nil {
		m.err = err.Error()
		return
	}
	m.tags = tags
	if len(tags) == 0 {
		m.cursor = 0
	} else if m.cursor >= len(tags) {
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

func (m model) onBack() bool { return m.cursor == len(m.tags) }

func (m model) page() int {
	if m.cursor >= len(m.tags) {
		if len(m.tags) == 0 {
			return 0
		}
		return (len(m.tags) - 1) / releasesPageSize
	}
	return m.cursor / releasesPageSize
}

func (m model) pageBounds() (int, int) {
	start := m.page() * releasesPageSize
	end := min(start+releasesPageSize, len(m.tags))
	return start, end
}

func (m model) pageCount() int {
	if len(m.tags) == 0 {
		return 1
	}
	return (len(m.tags)-1)/releasesPageSize + 1
}

func (m model) movePage(delta int) model {
	if len(m.tags) == 0 {
		return m
	}
	page := m.page() + delta
	if page < 0 || page >= m.pageCount() {
		return m
	}
	m.cursor = page * releasesPageSize
	m.detailOffset = 0
	m.status = ""
	return m
}

func (m model) detailLines() []string {
	if _, ok := m.selected(); !ok {
		return nil
	}
	return strings.Split(m.notesFor(m.cursor), "\n")
}

func (m model) maxDetailOffset() int {
	lines := m.detailLines()
	if len(lines) <= detailPaneLines {
		return 0
	}
	return len(lines) - detailPaneLines
}

func (m model) moveDetail(delta int) model {
	m.detailOffset = max(0, min(m.detailOffset+delta, m.maxDetailOffset()))
	m.previewBack = false
	m.previewConfirm = false
	return m
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	if m.mode == confirmDelete {
		switch key.String() {
		case "ctrl+c":
			m.quit = true
			m.killed = true
			return m, tea.Quit
		case "y", "Y":
			m = m.doDelete()
		case "n", "N", "esc", "q":
			m.mode = browse
			m.status = i18n.T(i18n.ReleasesDeleteCancelled)
		}
		return m, nil
	}

	switch key.String() {
	case "ctrl+c":
		m.quit = true
		m.killed = true
		return m, tea.Quit
	case "q":
		m.quit = true
		return m, tea.Quit
	case "esc", "tab":
		if m.mode == preview {
			m.mode = browse
			m.detailOffset = 0
			m.previewBack = false
			m.previewConfirm = false
			return m, nil
		}
		m.quit = true
		return m, tea.Quit
	}

	if m.mode == preview {
		switch key.String() {
		case "b", "h", "left":
			m.mode = browse
			m.detailOffset = 0
			m.previewBack = false
			m.previewConfirm = false
		case "up", "k":
			if m.previewBack {
				m.previewBack = false
				if m.selectable {
					m.previewConfirm = true
				}
			} else if m.previewConfirm {
				m.previewConfirm = false
			} else {
				m = m.moveDetail(-1)
			}
		case "down", "j":
			if m.detailOffset >= m.maxDetailOffset() {
				if m.selectable && !m.previewConfirm {
					m.previewConfirm = true
					m.previewBack = false
				} else {
					m.previewConfirm = false
					m.previewBack = true
				}
			} else {
				m = m.moveDetail(1)
			}
		case "pgup", "p":
			m = m.moveDetail(-detailPaneLines)
		case "pgdown", "n", " ":
			if m.detailOffset >= m.maxDetailOffset() {
				if m.selectable && !m.previewConfirm {
					m.previewConfirm = true
					m.previewBack = false
				} else {
					m.previewConfirm = false
					m.previewBack = true
				}
			} else {
				m = m.moveDetail(detailPaneLines)
			}
		case "enter":
			if m.previewBack {
				m.mode = browse
				m.detailOffset = 0
				m.previewBack = false
				m.previewConfirm = false
				return m, nil
			}
			if m.previewConfirm && m.selectable {
				m.confirmed = true
				return m, tea.Quit
			}
		}
		return m, nil
	}

	switch key.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
			m.detailOffset = 0
			m.status = ""
		}
	case "down", "j":
		if m.cursor < len(m.tags) {
			m.cursor++
			m.detailOffset = 0
			m.status = ""
		}
	case "l", "right":
		if _, ok := m.selected(); ok {
			m.mode = preview
			m.detailOffset = 0
		}
	case "n", "pgdown":
		m = m.movePage(1)
	case "p", "pgup":
		m = m.movePage(-1)
	case "enter", " ":
		if m.onBack() {
			m.quit = true
			return m, tea.Quit
		}
		if _, ok := m.selected(); ok {
			m.picked = m.cursor
			m.mode = preview
			m.detailOffset = 0
		}
	case "d", "x":
		if !m.selectable {
			if _, ok := m.selected(); ok {
				m.mode = confirmDelete
				m.status = ""
			}
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

	changelogRemoved, err := Delete(m.repo, m.changelogPath, tag.Name)
	if err != nil {
		var cErr *ChangelogError
		if errors.As(err, &cErr) {
			m.err = i18n.T(i18n.ReleasesChangelogWriteError, cErr.Err)
			m.reload()
		} else {
			m.err = err.Error()
		}
		return m
	}

	removed := i18n.T(i18n.ReleasesRemovedTag)
	if changelogRemoved {
		removed = i18n.T(i18n.ReleasesRemovedTagAndChangelog)
	}
	m.reload()
	m.detailOffset = 0
	m.status = i18n.T(i18n.ReleasesDeleted, tag.Name, removed)
	return m
}

// metaFor is the "<date> · <time> · N commits" line under the version.
func metaFor(tag gitrepo.TagInfo, commits int) string {
	when := tag.DateTime
	if when == "" {
		when = tag.Date
	}
	return ui.ReleaseMeta(when, commits)
}

// notesFor returns the text shown in the detail pane for the tag at idx. It
// rebuilds the notes from the commits that landed in that version so each line
// carries its commit hash; the changelog section and tag message are fallbacks.
func (m model) commitsFor(idx int) int {
	from := ""
	if idx+1 < len(m.tags) {
		from = m.tags[idx+1].Name
	}
	raw, err := m.repo.CommitsBetween(from, m.tags[idx].Name)
	if err != nil {
		return 0
	}
	return len(raw)
}

func (m model) rangeFor(idx int) (from, to string) {
	to = m.tags[idx].Name
	if idx+1 < len(m.tags) {
		from = m.tags[idx+1].Name // next entry is the previous (lower) version
	}
	return from, to
}

func (m model) notesFor(idx int) string {
	tag := m.tags[idx]

	from, to := m.rangeFor(idx)
	if raw, err := m.repo.CommitsBetween(from, to); err == nil && len(raw) > 0 {
		notes := changelog.Build(conventional.ParseMany(raw))
		return ui.ReleaseText(m.project, tag.Name, metaFor(tag, len(raw)), notes)
	}

	if s := changelog.ExtractSection(m.changelog, tag.Name); s != "" {
		return ui.ReleaseHeader(m.project, tag.Name, metaFor(tag, 0)) + "\n\n" + s
	}
	if msg, _ := m.repo.TagMessage(tag.Name); strings.TrimSpace(msg) != "" {
		return strings.TrimSpace(msg)
	}
	return ui.Dim.Render(i18n.T(i18n.ReleasesNoNotes))
}

const detailPaneLines = 18

var rowSel = lipgloss.NewStyle().Foreground(ui.Orange).Bold(true)

func (m model) View() string {
	if m.confirmed {
		return ""
	}
	if m.quit {
		return m.staticView()
	}

	var b strings.Builder
	b.WriteString(ui.Title.Render("⬢ "+i18n.T(i18n.ReleasesTitle)) + "\n\n")

	if m.err != "" {
		b.WriteString(ui.Warn.Render("✗ "+m.err) + "\n\n")
	}
	if len(m.tags) == 0 {
		b.WriteString(ui.Dim.Render(i18n.T(i18n.ReleasesEmpty)))
	} else if m.mode == preview {
		b.WriteString(m.previewView())
	} else {
		b.WriteString(m.tableView())
	}

	if m.mode == confirmDelete {
		tag, _ := m.selected()
		b.WriteString("\n" + ui.Warn.Render(i18n.T(i18n.ReleasesDeleteConfirm, tag.Name)))
	} else {
		if m.status != "" {
			b.WriteString("\n" + ui.Ok.Render("✓ ") + m.status)
		}
		sep := "\n\n"
		if strings.HasSuffix(b.String(), "\n") {
			sep = "\n"
		}
		b.WriteString(sep)
		if m.mode == preview && m.selectable {
			confirmLabel := i18n.T(i18n.ReleasesHintConfirm)
			confirmMarker := "  "
			if m.previewConfirm {
				confirmMarker = "▸ "
				confirmLabel = ui.Key.Render(confirmLabel)
			}
			b.WriteString(confirmMarker + confirmLabel + "\n")
		}
		label := backHomeLabel
		if m.mode == preview {
			label = backLabel
		}
		marker := "  "
		if m.onBack() || (m.mode == preview && m.previewBack) {
			marker = "▸ "
			label = ui.Key.Render(label)
		}
		b.WriteString(marker + label)
		if m.mode == preview {
			b.WriteString("\n\n" + keyHint("↑/↓", i18n.T(i18n.ReleasesHintMove)) + keyHint("enter", i18n.T(i18n.ReleasesHintConfirm)) + keyHint("b/esc", i18n.T(i18n.ReleasesHintBack)))
		} else {
			footer := keyHint("↑/↓", i18n.T(i18n.ReleasesHintMove)) + keyHint("enter", i18n.T(i18n.ReleasesHintShowExit)) + keyHint("n/p", i18n.T(i18n.ReleasesHintPage))
			if !m.selectable {
				footer += keyHint("d", i18n.T(i18n.ReleasesHintDelete))
			}
			footer += keyHint("q", i18n.T(i18n.ReleasesHintBack))
			b.WriteString("\n\n" + footer)
		}
	}
	return b.String()
}

func (m model) tableView() string {
	start, end := m.pageBounds()
	rows := []string{
		fmt.Sprintf("╭─ %s %s", rowSel.Render("Releases"), strings.Repeat("─", 26)),
		fmt.Sprintf("│ %-10s %-10s %7s", i18n.T(i18n.ReleasesColumnVersion), i18n.T(i18n.ReleasesColumnDate), i18n.T(i18n.ReleasesColumnCommits)),
		ui.Dim.Render("│ ────────── ────────── ───────"),
	}
	for i := start; i < end; i++ {
		t := m.tags[i]
		mark := " "
		if i == m.cursor {
			mark = "›"
		}
		row := fmt.Sprintf("│ %s %-10s %-10s %7d", mark, t.Name, t.Date, m.commitsFor(i))
		if i == m.cursor {
			row = rowSel.Render(row)
		}
		rows = append(rows, row)
	}
	rows = append(rows, ui.Dim.Render(fmt.Sprintf("│ %s %d/%d", i18n.T(i18n.ReleasesPageLabel), m.page()+1, m.pageCount())))
	rows = append(rows, "╰"+strings.Repeat("─", 36))
	return strings.Join(rows, "\n") + "\n"
}

func (m model) previewView() string {
	lines := m.detailLines()
	if m.detailOffset > m.maxDetailOffset() {
		m.detailOffset = m.maxDetailOffset()
	}
	endLine := min(len(lines), m.detailOffset+detailPaneLines)
	rows := []string{fmt.Sprintf("╭─ %s %s", rowSel.Render("Preview"), strings.Repeat("─", 54))}
	if m.detailOffset > 0 {
		rows = append(rows, ui.Dim.Render("│ ↑ more"))
	}
	for _, line := range lines[m.detailOffset:endLine] {
		rows = append(rows, "│ "+line)
	}
	if endLine < len(lines) {
		rows = append(rows, ui.Dim.Render("│ ↓ more"))
	}
	rows = append(rows, "╰"+strings.Repeat("─", 64))
	return strings.Join(rows, "\n") + "\n"
}

// keyHint renders "<key> label" with the key highlighted, padded for a footer row.
func keyHint(key, label string) string {
	return ui.Key.Render(key) + ui.Dim.Render(" "+label+"   ")
}

// staticView is what remains in the scrollback after quitting. A soft q/esc
// back-out leaves no release list behind, so returning to the menu does not
// duplicate every version in the terminal history.
func (m model) staticView() string { return "" }

func changelogPath(repo *gitrepo.Repo, cfg config.Config) string {
	path := cfg.Release.ChangelogFile
	if path == "" {
		path = "CHANGELOG.md"
	}
	return repo.Root() + string(os.PathSeparator) + path
}

// Run opens the browser against repo and blocks until the user goes back.
func Run(repo *gitrepo.Repo, cfg config.Config) error {
	final, err := tea.NewProgram(newModel(repo, cfg.Project, changelogPath(repo, cfg))).Run()
	if err != nil {
		return err
	}
	if final.(model).killed {
		return ErrQuit
	}
	return nil
}

// RunSelect opens the shared release table and returns the release confirmed
// from its preview screen. Backing out returns ok=false.
func RunSelect(repo *gitrepo.Repo, cfg config.Config) (version string, ok bool, err error) {
	final, err := tea.NewProgram(newSelectModel(repo, cfg.Project, changelogPath(repo, cfg))).Run()
	if err != nil {
		return "", false, err
	}
	out := final.(model)
	if out.killed {
		return "", false, ErrQuit
	}
	if !out.confirmed {
		return "", false, nil
	}
	tag, selected := out.selected()
	return tag.Name, selected, nil
}
