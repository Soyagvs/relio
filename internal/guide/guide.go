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
	s1 := "The flow is:  code → commit → push → relio → version + CHANGELOG + tag. " +
		"Nothing is written until you confirm the preview, and Relio never pushes on its own unless you ask it to."
	if !ctx.HasRepo {
		s1 += "\nRun this inside a git repository to follow the steps below."
	}
	steps = append(steps, step{title: "What Relio does", body: s1})

	// 2
	if ctx.HasConfig {
		steps = append(steps, step{
			title: "Set up `.release.yaml`",
			body: fmt.Sprintf("Already set up (project: %s), so you can skip `relio init`. This is "+
				"Relio's own config at the repo root — not your package.json / pyproject.toml. "+
				"Configuration only, never secrets: changelog file, tag prefix, and optionally "+
				"version_files and hooks.", ctx.Project),
		})
	} else {
		s := step{
			title: "Set up `.release.yaml`",
			body: "Relio needs its own file, `.release.yaml`, at the repo root — separate from any " +
				"version file your language already has (package.json, pyproject.toml, …), and always " +
				"read from the project root no matter where you run relio from. Configuration only, " +
				"never secrets: changelog file, tag prefix, and optionally version_files (list those " +
				"language files here to keep them in sync) and hooks. Create it with `relio init`, or " +
				"hand-write a minimal one — just `project: <name>` works.",
		}
		if ctx.RunInit != nil {
			s.actionLabel, s.action = "run relio init now", ctx.RunInit
		}
		steps = append(steps, s)
	}

	// 3
	s3 := step{
		title: "Write Conventional Commits",
		body: "`feat:` bumps the minor; `fix:` / `perf:` / `refactor:` bump the patch; " +
			"`feat!:` or a `BREAKING CHANGE:` footer bumps the major. Commits without a type are ignored for versioning.",
	}
	if ctx.RunCheck != nil {
		s3.actionLabel, s3.action = "run relio check now", ctx.RunCheck
		s3.body += " Run `relio check` to see which of your commits qualify."
	}
	steps = append(steps, s3)

	// 4
	s4 := step{
		title: "See what's pending",
		body:  "`relio status` lists the unreleased commits and the version they suggest.",
	}
	if ctx.RunStatus != nil {
		s4.actionLabel, s4.action = "run relio status now", ctx.RunStatus
	}
	steps = append(steps, s4)

	// 5
	steps = append(steps, step{
		title: "Create the release",
		body: "Run `relio` with no arguments: you get a preview, then a small wizard " +
			"(Create / change the bump / cancel). On confirm it writes the CHANGELOG.md section, " +
			"commits it as `chore(release): vX.Y.Z`, and creates the annotated tag — nothing before the confirm.\n" +
			"`relio --rc` cuts a release candidate you can iterate on; running `relio` again on an rc finalizes it.",
	})

	// 6
	s6 := "Push with `git push --follow-tags`. Or `relio --publish` to push and create the GitHub Release " +
		"with the changelog notes as its body — that needs a GitHub token (GITHUB_TOKEN / GH_TOKEN / `gh auth login`)."
	if !ctx.HasToken {
		s6 += "\nNo GitHub token is set yet."
	}
	steps = append(steps, step{title: "Get it out", body: s6})

	// 7
	steps = append(steps, step{
		title: "Optional extras",
		body: "`relio post` prints announcement text for socials. `relio image` renders a PNG release card. " +
			"`.release.yaml` `version_files:` writes the new version into package.json / pyproject.toml / …. " +
			"`.release.yaml` `release.hooks.before` / `.after` run shell commands around the release.",
	})

	// 8
	steps = append(steps, step{
		title: "You're set",
		body: "Happy path:  relio init → write feat:/fix: commits → relio status → relio → git push --follow-tags.\n" +
			"See `relio help` for every command and flag, and the README for the full `.release.yaml` reference.",
	})

	return steps
}

const happyPath = "relio init → write feat:/fix: commits → relio status → relio → git push --follow-tags"

// PrintPlain writes every step as text, with no actions and no prompts.
func PrintPlain(w io.Writer, ctx Context) error {
	fmt.Fprintln(w, "Relio — guide")
	fmt.Fprintln(w)
	for i, s := range buildSteps(ctx) {
		fmt.Fprintf(w, "%d. %s\n", i+1, s.title)
		for _, line := range wrapLines(s.body, 76) {
			fmt.Fprintf(w, "   %s\n", line)
		}
		fmt.Fprintln(w)
	}
	fmt.Fprintln(w, "The happy path:  "+happyPath)
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
	b.WriteString(ui.Dim.Render(fmt.Sprintf("Step %d of %d", m.i+1, len(m.steps))) + "\n\n")
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
		return "[y] " + m.steps[m.i].actionLabel + " · enter skip · ← back · q quit"
	}
	return "enter continue · ← back · q quit"
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
