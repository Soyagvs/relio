package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/soyagvs/relio/internal/i18n"
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

// TestNewGuideCmdShortLocalizesAtConstructionTime proves newGuideCmd()'s
// Short resolves through the active i18n catalog at construction time. It
// reuses cmd/help.go's HelpCmdGuideDesc key — byte-identical to the
// original hardcoded English literal — instead of declaring a duplicate.
func TestNewGuideCmdShortLocalizesAtConstructionTime(t *testing.T) {
	prev := i18n.Current()
	t.Cleanup(func() { i18n.SetLanguage(prev) })

	i18n.SetLanguage("en")
	cmdEN := newGuideCmd(&releaseFlags{})
	i18n.SetLanguage("es")
	cmdES := newGuideCmd(&releaseFlags{})

	if cmdES.Short == cmdEN.Short {
		t.Errorf("guide Short unchanged across languages: %q", cmdES.Short)
	}
}

// TestGoldenEnglishGuideCmdUnchanged pins cmd/guide.go's hardcoded English
// Short literal against i18n.T() under the default "en" language.
func TestGoldenEnglishGuideCmdUnchanged(t *testing.T) {
	prev := i18n.Current()
	if prev != "en" {
		i18n.SetLanguage("en")
	}
	t.Cleanup(func() { i18n.SetLanguage(prev) })

	c := newGuideCmd(&releaseFlags{})
	if c.Short != "Walk through the whole release flow step by step" {
		t.Errorf("Short = %q", c.Short)
	}
}
