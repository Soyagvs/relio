package guide

import (
	"bytes"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestBuildStepsAlwaysEight(t *testing.T) {
	cases := []Context{
		{},
		{HasRepo: true},
		{HasRepo: true, HasConfig: true, Project: "widget", HasToken: true},
		{HasRepo: true, RunInit: func() error { return nil }, RunCheck: func() error { return nil }, RunStatus: func() error { return nil }},
	}
	for i, ctx := range cases {
		if n := len(buildSteps(ctx)); n != 8 {
			t.Errorf("case %d: buildSteps = %d steps, want 8", i, n)
		}
	}
}

func TestPrintPlainHasEveryTitleAndHappyPath(t *testing.T) {
	var buf bytes.Buffer
	if err := PrintPlain(&buf, Context{HasRepo: true}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, title := range []string{
		"What Relio does",
		"Set up",
		"Conventional Commits",
		"what's pending",
		"Create the release",
		"Get it out",
		"Optional extras",
		"You're set",
	} {
		if !strings.Contains(out, title) {
			t.Errorf("PrintPlain missing step title %q:\n%s", title, out)
		}
	}
	if !strings.Contains(out, "The happy path:  relio init → write feat:/fix: commits → relio status → relio → git push --follow-tags") {
		t.Errorf("PrintPlain missing/!wrong happy-path line:\n%s", out)
	}
}

func TestPrintPlainNeverPrompts(t *testing.T) {
	cases := []Context{
		{},
		{HasRepo: true},
		{HasRepo: true, RunInit: func() error { return nil }, RunCheck: func() error { return nil }, RunStatus: func() error { return nil }},
		{HasRepo: true, HasConfig: true, Project: "widget"},
	}
	for i, ctx := range cases {
		var buf bytes.Buffer
		if err := PrintPlain(&buf, ctx); err != nil {
			t.Fatalf("case %d: %v", i, err)
		}
		if strings.Contains(buf.String(), "[y]") {
			t.Errorf("case %d: PrintPlain emitted a [y] prompt:\n%s", i, buf.String())
		}
	}
}

func TestPrintPlainWithConfigNamesProjectAndSkipsInitOffer(t *testing.T) {
	var buf bytes.Buffer
	if err := PrintPlain(&buf, Context{HasRepo: true, HasConfig: true, Project: "widget"}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "widget") {
		t.Errorf("PrintPlain should name the configured project:\n%s", out)
	}
	if strings.Contains(out, "run relio init") {
		t.Errorf("PrintPlain should not offer `relio init` when config exists:\n%s", out)
	}
}

func TestStepperAdvancesAndRunsAction(t *testing.T) {
	ran := false
	m := teaModel{steps: buildSteps(Context{HasRepo: true, RunInit: func() error { ran = true; return nil }})}

	n, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = n.(teaModel)
	if m.i != 1 {
		t.Fatalf("enter did not advance from step 1: i=%d", m.i)
	}

	n, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	m = n.(teaModel)
	if !ran {
		t.Error("y did not invoke the step action")
	}
	if m.i != 2 {
		t.Errorf("y did not advance past the action step: i=%d", m.i)
	}
}
