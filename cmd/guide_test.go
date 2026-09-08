package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestNewGuideCmdShape(t *testing.T) {
	c := newGuideCmd(&releaseFlags{})

	if c.Use != "guide" {
		t.Errorf("Use = %q, want %q", c.Use, "guide")
	}
	if c.Short == "" {
		t.Error("guide command needs a Short description")
	}
	if c.RunE == nil {
		t.Error("guide command needs a RunE")
	}
	if err := c.Args(c, []string{"extra"}); err == nil {
		t.Error("guide should reject positional args")
	}

	root := NewRootCmd()
	found := false
	for _, sub := range root.Commands() {
		if sub.Name() == "guide" {
			found = true
		}
	}
	if !found {
		t.Error("guide is not registered on the root command")
	}
}

func TestRunGuidePlainRendersAllSteps(t *testing.T) {
	dir := t.TempDir() // not a git repo — the guide still explains the flow

	c := newGuideCmd(&releaseFlags{dir: dir})
	var out bytes.Buffer
	c.SetOut(&out)
	c.SetErr(&out)

	if err := c.RunE(c, nil); err != nil {
		t.Fatalf("guide: %v", err)
	}
	for _, want := range []string{"Relio — guide", "What Relio does", "You're set", "The happy path:"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("guide plain output missing %q:\n%s", want, out.String())
		}
	}
	if strings.Contains(out.String(), "[y]") {
		t.Errorf("guide plain output should carry no prompts:\n%s", out.String())
	}
}
