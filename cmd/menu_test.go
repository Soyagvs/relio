package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/soyagvs/relio/internal/config"
	"github.com/soyagvs/relio/internal/pick"
)

func TestRunMenuSetupOnExistingConfig(t *testing.T) {
	_, dir := initTempRepo(t)
	if err := config.Default("proj").Save(dir); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&buf)
	cmd.SetIn(strings.NewReader(""))

	if err := runMenuSetup(cmd, &releaseFlags{dir: dir}); err != nil {
		t.Fatalf("runMenuSetup: %v", err)
	}
	if !strings.Contains(buf.String(), "already exists") {
		t.Errorf("Setup on an existing config = %q, want it to mention 'already exists'", buf.String())
	}
}

func TestWaitMenuBackPrintsSeparatorClearsAndRunsBackPicker(t *testing.T) {
	oldRunPicker := runPicker
	t.Cleanup(func() { runPicker = oldRunPicker })

	var gotTitle string
	var gotItems []pick.Item
	runPicker = func(title string, items []pick.Item) (string, bool, error) {
		gotTitle = title
		gotItems = items
		return "", false, nil
	}

	var buf bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&buf)

	if err := waitMenuBack(cmd); err != nil {
		t.Fatalf("waitMenuBack: %v", err)
	}
	if buf.String() != "\n\x1b[2J\x1b[H" {
		t.Fatalf("waitMenuBack output = %q, want blank line then clear screen", buf.String())
	}
	if gotTitle != "" || gotItems != nil {
		t.Fatalf("runPicker got title %q items %#v, want empty back picker", gotTitle, gotItems)
	}
}

func TestWaitMenuBackPropagatesHardQuit(t *testing.T) {
	oldRunPicker := runPicker
	t.Cleanup(func() { runPicker = oldRunPicker })

	runPicker = func(string, []pick.Item) (string, bool, error) {
		return "", false, pick.ErrQuit
	}

	var buf bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&buf)

	if err := waitMenuBack(cmd); err != pick.ErrQuit {
		t.Fatalf("waitMenuBack error = %v, want pick.ErrQuit", err)
	}
	if buf.String() != "\n" {
		t.Fatalf("waitMenuBack output on hard quit = %q, want only blank line", buf.String())
	}
}

func TestRunMenuAuthBackChoiceSkipsMenuBackWait(t *testing.T) {
	oldRunPicker := runPicker
	t.Cleanup(func() { runPicker = oldRunPicker })

	runPicker = func(string, []pick.Item) (string, bool, error) {
		return "", false, nil
	}

	wait, err := runMenuAuth(&cobra.Command{})
	if err != nil {
		t.Fatalf("runMenuAuth: %v", err)
	}
	if wait {
		t.Fatal("runMenuAuth wait = true after nested back choice, want false")
	}
}

func TestRunMenuReleaseBackStepsToPreviousPicker(t *testing.T) {
	_, dir := initTempRepo(t)
	if err := config.Default("proj").Save(dir); err != nil {
		t.Fatal(err)
	}

	oldRunPicker := runPicker
	t.Cleanup(func() { runPicker = oldRunPicker })

	responses := []struct {
		value  string
		chosen bool
	}{
		{value: "final", chosen: true},
		{chosen: false},
		{chosen: false},
	}
	var titles []string
	runPicker = func(title string, items []pick.Item) (string, bool, error) {
		titles = append(titles, title)
		if len(responses) == 0 {
			t.Fatalf("unexpected picker %q", title)
		}
		next := responses[0]
		responses = responses[1:]
		return next.value, next.chosen, nil
	}

	wait, err := runMenuRelease(&cobra.Command{}, &releaseFlags{dir: dir})
	if err != nil {
		t.Fatalf("runMenuRelease: %v", err)
	}
	if wait {
		t.Fatal("runMenuRelease wait = true after backing out to menu, want false")
	}
	want := []string{"Release type", "Publish to GitHub?", "Release type"}
	if strings.Join(titles, "|") != strings.Join(want, "|") {
		t.Fatalf("picker titles = %#v, want %#v", titles, want)
	}
}

func TestRunMenuReleaseBackFromEditStepsToPublish(t *testing.T) {
	_, dir := initTempRepo(t)
	if err := config.Default("proj").Save(dir); err != nil {
		t.Fatal(err)
	}

	oldRunPicker := runPicker
	t.Cleanup(func() { runPicker = oldRunPicker })

	responses := []struct {
		value  string
		chosen bool
	}{
		{value: "final", chosen: true},
		{value: "local", chosen: true},
		{chosen: false},
		{chosen: false},
		{chosen: false},
	}
	var titles []string
	runPicker = func(title string, items []pick.Item) (string, bool, error) {
		titles = append(titles, title)
		if len(responses) == 0 {
			t.Fatalf("unexpected picker %q", title)
		}
		next := responses[0]
		responses = responses[1:]
		return next.value, next.chosen, nil
	}

	wait, err := runMenuRelease(&cobra.Command{}, &releaseFlags{dir: dir})
	if err != nil {
		t.Fatalf("runMenuRelease: %v", err)
	}
	if wait {
		t.Fatal("runMenuRelease wait = true after backing out to menu, want false")
	}
	want := []string{"Release type", "Publish to GitHub?", "Edit the notes first?", "Publish to GitHub?", "Release type"}
	if strings.Join(titles, "|") != strings.Join(want, "|") {
		t.Fatalf("picker titles = %#v, want %#v", titles, want)
	}
}
