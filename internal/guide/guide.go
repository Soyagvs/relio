// Package guide is `relio guide`: a step-by-step walkthrough of the whole
// release flow. It renders either as an interactive Bubble Tea stepper (when
// stdin/stdout are a TTY) or as plain text.
package guide

import (
	"errors"
	"fmt"
	"io"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/soyagvs/relio/internal/i18n"
	"github.com/soyagvs/relio/internal/ui"
)

// Context tailors the walkthrough to the current repo. The Run* callbacks are
// optional "do it now" actions the interactive stepper may invoke; each is
// nil-safe and only wired when it makes sense (a repo, and — for check/status —
// a loadable config).
type Context struct {
	HasRepo   bool
	HasConfig bool
	Project   string // "" when there is no config
	HasToken  bool   // ghrelease.Token() != ""

	RunInit   func() error // runs `relio init`
	RunCheck  func() error // runs `relio check`
	RunStatus func() error // runs `relio status`
}

type step struct {
	title       string
	body        string
	actionLabel string       // "" when the step has no action
	action      func() error // nil-safe
}

// buildSteps returns the eight walkthrough steps, tailored by ctx. The count is
// always eight; only the wording and the attached actions vary.
func buildSteps(ctx Context) []step {
	steps := make([]step, 0, 8)

	// 1
	s1 := i18n.T(i18n.GuideStep1Body)
	if !ctx.HasRepo {
		s1 += "\n" + i18n.T(i18n.GuideStep1NoRepoHint)
	}
	steps = append(steps, step{title: i18n.T(i18n.GuideStep1Title), body: s1})

	// 2
	if ctx.HasConfig {
		steps = append(steps, step{
			title: i18n.T(i18n.GuideStep2Title),
			body:  i18n.T(i18n.GuideStep2ConfiguredBody, ctx.Project),
		})
	} else {
		s := step{
			title: i18n.T(i18n.GuideStep2Title),
			body:  i18n.T(i18n.GuideStep2UnconfiguredBody),
		}
		if ctx.RunInit != nil {
			s.actionLabel, s.action = i18n.T(i18n.GuideActionRunInit), ctx.RunInit
		}
		steps = append(steps, s)
	}

	// 3
	s3 := step{
		title: i18n.T(i18n.GuideStep3Title),
		body:  i18n.T(i18n.GuideStep3Body),
	}
	if ctx.RunCheck != nil {
		s3.actionLabel, s3.action = i18n.T(i18n.GuideActionRunCheck), ctx.RunCheck
		s3.body += " " + i18n.T(i18n.GuideStep3CheckHint)
	}
	steps = append(steps, s3)

	// 4
	s4 := step{
		title: i18n.T(i18n.GuideStep4Title),
		body:  i18n.T(i18n.GuideStep4Body),
	}
	if ctx.RunStatus != nil {
		s4.actionLabel, s4.action = i18n.T(i18n.GuideActionRunStatus), ctx.RunStatus
	}
	steps = append(steps, s4)

	// 5
	steps = append(steps, step{
		title: i18n.T(i18n.GuideStep5Title),
		body:  i18n.T(i18n.GuideStep5Body),
	})

	// 6
	s6 := i18n.T(i18n.GuideStep6Body)
	if !ctx.HasToken {
		s6 += "\n" + i18n.T(i18n.GuideStep6NoTokenHint)
	}
	steps = append(steps, step{title: i18n.T(i18n.GuideStep6Title), body: s6})

	// 7
	steps = append(steps, step{
		title: i18n.T(i18n.GuideStep7Title),
		body:  i18n.T(i18n.GuideStep7Body),
	})

	// 8
	steps = append(steps, step{
		title: i18n.T(i18n.GuideStep8Title),
		body:  i18n.T(i18n.GuideStep8Body, i18n.T(i18n.GuideHappyPath)),
	})

	return steps
}

// PrintPlain writes every step as text, with no actions and no prompts.
func PrintPlain(w io.Writer, ctx Context) error {
	fmt.Fprintln(w, i18n.T(i18n.GuidePlainHeader))
	fmt.Fprintln(w)
	for i, s := range buildSteps(ctx) {
		fmt.Fprintf(w, "%d. %s\n", i+1, s.title)
		for _, line := range wrapLines(s.body, 76) {
			fmt.Fprintf(w, "   %s\n", line)
		}
		fmt.Fprintln(w)
	}
	fmt.Fprintln(w, i18n.T(i18n.GuidePlainFooter, i18n.T(i18n.GuideHappyPath)))
	return nil
}

// wrapLines honours explicit newlines in s, then word-wraps each paragraph to
// width columns.
func wrapLines(s string, width int) []string {
	var out []string
	for _, para := range strings.Split(s, "\n") {
		words := strings.Fields(para)
		if len(words) == 0 {
			out = append(out, "")
			continue
		}
		line := words[0]
		for _, wd := range words[1:] {
			if len(line)+1+len(wd) > width {
				out = append(out, line)
				line = wd
				continue
			}
			line += " " + wd
		}
		out = append(out, line)
	}
	return out
}

// ErrQuit is returned by Run when the user hard-quits with ctrl+c, rather than
// leaving the walkthrough with q/esc (which returns nil).
var ErrQuit = errors.New("guide: quit")

// teaModel is the interactive stepper.
type teaModel struct {
	steps  []step
	i      int
	msg    string // transient line under the body (an action's error)
	done   bool
	killed bool // hard-quit with ctrl+c
}

func (m teaModel) Init() tea.Cmd { return nil }

func (m teaModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	k, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	cur := m.steps[m.i]
	switch k.String() {
	case "ctrl+c":
		m.done = true
		m.killed = true
		return m, tea.Quit
	case "q", "esc":
		m.done = true
		return m, tea.Quit
	case "left", "h":
		if m.i > 0 {
			m.i--
			m.msg = ""
		}
	case "y", "Y":
		if cur.action != nil {
			m.msg = ""
			if err := cur.action(); err != nil {
				m.msg = "error: " + err.Error()
			}
			m.i++
		}
	case "enter", "right", "l", " ", "n", "N":
		m.msg = ""
		m.i++
	}
	if m.i >= len(m.steps) {
		m.done = true
		return m, tea.Quit
	}
	return m, nil
}

func (m teaModel) View() string {
	if m.done || m.i >= len(m.steps) {
		return ""
	}
	s := m.steps[m.i]

	var b strings.Builder
	b.WriteString(ui.Dim.Render(i18n.T(i18n.GuideStepCounter, m.i+1, len(m.steps))) + "\n\n")
	b.WriteString(ui.Title.Render(s.title) + "\n\n")
	for _, line := range wrapLines(s.body, 76) {
		b.WriteString("  " + line + "\n")
	}
	if m.msg != "" {
		b.WriteString("\n  " + ui.Warn.Render(m.msg) + "\n")
	}
	b.WriteString("\n" + ui.Dim.Render(m.footer()))
	return b.String()
}

func (m teaModel) footer() string {
	if m.steps[m.i].action != nil {
		return i18n.T(i18n.GuideFooterWithAction, m.steps[m.i].actionLabel)
	}
	return i18n.T(i18n.GuideFooterNoAction)
}

// Run shows the interactive stepper and blocks until the user leaves it.
func Run(ctx Context) error {
	final, err := tea.NewProgram(teaModel{steps: buildSteps(ctx)}).Run()
	if err != nil {
		return err
	}
	if final.(teaModel).killed {
		return ErrQuit
	}
	return nil
}
