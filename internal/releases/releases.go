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
	confirmDelete
)

type focus int

const (
	focusTable focus = iota
	focusDetail
)

type model struct {
	repo          repoPort
	project       string
	changelogPath string
	changelog     string

	tags         []gitrepo.TagInfo
	cursor       int
	mode         mode
	status       string
	err          string
	quit         bool
	killed       bool // hard-quit with ctrl+c
	picked       int  // retained for older static output paths; -1 = none
	focus        focus
	detailOffset int
}

const backLabel = "<- Back"
const releasesPageSize = 10

func newModel(repo repoPort, project, changelogPath string) model {
	m := model{repo: repo, project: project, changelogPath: changelogPath, picked: -1}
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
		if m.focus == focusDetail {
			m.focus = focusTable
			return m, nil
		}
		m.quit = true
		return m, tea.Quit
	}

	if m.focus == focusDetail {
		switch key.String() {
		case "h", "left":
			m.focus = focusTable
		case "up", "k":
			m = m.moveDetail(-1)
		case "down", "j":
			m = m.moveDetail(1)
		case "pgup", "p":
			m = m.moveDetail(-detailPaneLines)
		case "pgdown", "n", " ":
			m = m.moveDetail(detailPaneLines)
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
			m.focus = focusDetail
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

var (
	rowSel      = lipgloss.NewStyle().Foreground(ui.Orange).Bold(true)
	detailFocus = lipgloss.NewStyle().Foreground(ui.Orange).Bold(true)
)

func (m model) View() string {
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
	} else {
		b.WriteString(m.splitView())
	}

	if m.mode == confirmDelete {
		tag, _ := m.selected()
		b.WriteString("\n" + ui.Warn.Render(i18n.T(i18n.ReleasesDeleteConfirm, tag.Name)))
	} else {
		if m.status != "" {
			b.WriteString("\n" + ui.Ok.Render("✓ ") + m.status)
		}
		label := backLabel
		marker := "  "
		if m.onBack() {
			marker = "▸ "
			label = ui.Key.Render(label)
		}
		sep := "\n\n"
		if strings.HasSuffix(b.String(), "\n") {
			sep = "\n"
		}
		b.WriteString(sep + marker + label)
		if m.focus == focusDetail {
			b.WriteString("\n\n" + keyHint("↑/↓", i18n.T(i18n.ReleasesHintMove)) + keyHint("h/esc", i18n.T(i18n.ReleasesHintBack)) + keyHint("q", i18n.T(i18n.ReleasesHintBack)))
		} else {
			b.WriteString("\n\n" + keyHint("↑/↓", i18n.T(i18n.ReleasesHintMove)) + keyHint("l", i18n.T(i18n.ReleasesHintShowExit)) + keyHint("n/p", i18n.T(i18n.ReleasesHintPage)) +
				keyHint("d", i18n.T(i18n.ReleasesHintDelete)) + keyHint("q", i18n.T(i18n.ReleasesHintBack)))
		}
	}
	return b.String()
}

func (m model) splitView() string {
	start, end := m.pageBounds()
	leftTitle := "Releases"
	if m.focus == focusTable {
		leftTitle = rowSel.Render(leftTitle)
	}
	left := []string{
		fmt.Sprintf("╭─ %s %s", leftTitle, strings.Repeat("─", 26)),
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
		if i == m.cursor && m.focus == focusTable {
			row = rowSel.Render(row)
		}
		left = append(left, row)
	}
	left = append(left, ui.Dim.Render(fmt.Sprintf("│ %s %d/%d", i18n.T(i18n.ReleasesPageLabel), m.page()+1, m.pageCount())))
	left = append(left, "╰"+strings.Repeat("─", 36))

	rightTitle := "Summary"
	if m.focus == focusDetail {
		rightTitle = detailFocus.Render(rightTitle)
	}
	right := []string{fmt.Sprintf("╭─ %s %s", rightTitle, strings.Repeat("─", 54))}
	lines := m.detailLines()
	if m.detailOffset > m.maxDetailOffset() {
		m.detailOffset = m.maxDetailOffset()
	}
	endLine := min(len(lines), m.detailOffset+detailPaneLines)
	if m.detailOffset > 0 {
		right = append(right, ui.Dim.Render("│ ↑ more"))
	}
	for _, line := range lines[m.detailOffset:endLine] {
		right = append(right, "│ "+line)
	}
	if endLine < len(lines) {
		right = append(right, ui.Dim.Render("│ ↓ more"))
	}
	right = append(right, "╰"+strings.Repeat("─", 64))

	rows := max(len(left), len(right))
	var b strings.Builder
	for i := 0; i < rows; i++ {
		l, r := "", ""
		if i < len(left) {
			l = left[i]
		}
		if i < len(right) {
			r = right[i]
		}
		b.WriteString(padRight(l, 38) + "  " + r + "\n")
	}
	return b.String()
}

func padRight(s string, width int) string {
	if n := lipgloss.Width(s); n < width {
		return s + strings.Repeat(" ", width-n)
	}
	return s
}

// keyHint renders "<key> label" with the key highlighted, padded for a footer row.
func keyHint(key, label string) string {
	return ui.Key.Render(key) + ui.Dim.Render(" "+label+"   ")
}

// staticView is what remains in the scrollback after quitting. A soft q/esc
// back-out leaves no release list behind, so returning to the menu does not
// duplicate every version in the terminal history.
func (m model) staticView() string { return "" }

// Run opens the browser against repo and blocks until the user goes back.
func Run(repo *gitrepo.Repo, cfg config.Config) error {
	path := cfg.Release.ChangelogFile
	if path == "" {
		path = "CHANGELOG.md"
	}
	m := newModel(repo, cfg.Project, repo.Root()+string(os.PathSeparator)+path)
	final, err := tea.NewProgram(m).Run()
	if err != nil {
		return err
	}
	if final.(model).killed {
		return ErrQuit
	}
	return nil
}
