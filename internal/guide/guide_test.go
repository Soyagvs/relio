package guide

import (
	"bytes"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/soyagvs/relio/internal/i18n"
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

func TestStepperQLeavesWithoutKilling(t *testing.T) {
	m := teaModel{steps: buildSteps(Context{})}
	n, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	m = n.(teaModel)
	if !m.done {
		t.Fatal("q should set done")
	}
	if m.killed {
		t.Error("q should not set killed — it is a soft back-out, not a hard quit")
	}
}

func TestStepperCtrlCKills(t *testing.T) {
	m := teaModel{steps: buildSteps(Context{})}
	n, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	m = n.(teaModel)
	if !m.done || !m.killed {
		t.Fatalf("ctrl+c should set done and killed, got done=%v killed=%v", m.done, m.killed)
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

// TestGoldenEnglishDefaultUnchanged pins guide.go's own chrome and all eight
// step titles/bodies against i18n.T() under the default "en" language:
// converting guide.go's literals to i18n.T() calls MUST NOT change a single
// byte of English output. This is the RED/refactor safety net for the
// guide.go -> i18n.T() conversion (slice 5c).
func TestGoldenEnglishDefaultUnchanged(t *testing.T) {
	prev := i18n.Current()
	if prev != "en" {
		i18n.SetLanguage("en")
	}
	t.Cleanup(func() { i18n.SetLanguage(prev) })

	// Step 1 — no-repo variant included, byte-identical to the historical
	// hardcoded strings.
	steps := buildSteps(Context{})
	if steps[0].title != "What Relio does" {
		t.Errorf("step1 title = %q", steps[0].title)
	}
	wantStep1 := "The flow is:  code → commit → push → relio → version + CHANGELOG + tag. " +
		"Nothing is written until you confirm the preview, and Relio never pushes on its own unless you ask it to.\n" +
		"Run this inside a git repository to follow the steps below."
	if steps[0].body != wantStep1 {
		t.Errorf("step1 body = %q, want %q", steps[0].body, wantStep1)
	}

	// Step 2 — both variants, byte-identical to the historical hardcoded
	// strings.
	configuredSteps := buildSteps(Context{HasRepo: true, HasConfig: true, Project: "widget"})
	wantStep2Configured := "Already set up (project: widget), so you can skip `relio init`. This is " +
		"Relio's own config at the repo root — not your package.json / pyproject.toml. " +
		"Configuration only, never secrets: changelog file, tag prefix, and optionally " +
		"version_files and hooks."
	if configuredSteps[1].body != wantStep2Configured {
		t.Errorf("step2 (configured) body = %q, want %q", configuredSteps[1].body, wantStep2Configured)
	}
	if configuredSteps[1].title != "Set up `.release.yaml`" {
		t.Errorf("step2 title = %q", configuredSteps[1].title)
	}

	unconfiguredSteps := buildSteps(Context{HasRepo: true, RunInit: func() error { return nil }})
	wantStep2Unconfigured := "Relio needs its own file, `.release.yaml`, at the repo root — separate from any " +
		"version file your language already has (package.json, pyproject.toml, …), and always " +
		"read from the project root no matter where you run relio from. Configuration only, " +
		"never secrets: changelog file, tag prefix, and optionally version_files (list those " +
		"language files here to keep them in sync) and hooks. Create it with `relio init`, or " +
		"hand-write a minimal one — just `project: <name>` works."
	if unconfiguredSteps[1].body != wantStep2Unconfigured {
		t.Errorf("step2 (unconfigured) body = %q, want %q", unconfiguredSteps[1].body, wantStep2Unconfigured)
	}
	if unconfiguredSteps[1].actionLabel != "run relio init now" {
		t.Errorf("step2 actionLabel = %q, want %q", unconfiguredSteps[1].actionLabel, "run relio init now")
	}

	// Step 3 — with and without the check hint.
	plain3 := buildSteps(Context{})
	wantStep3 := "`feat:` bumps the minor; `fix:` / `perf:` / `refactor:` bump the patch; " +
		"`feat!:` or a `BREAKING CHANGE:` footer bumps the major. Commits without a type are ignored for versioning."
	if plain3[2].body != wantStep3 {
		t.Errorf("step3 body = %q, want %q", plain3[2].body, wantStep3)
	}
	withCheck := buildSteps(Context{RunCheck: func() error { return nil }})
	wantStep3WithCheck := wantStep3 + " Run `relio check` to see which of your commits qualify."
	if withCheck[2].body != wantStep3WithCheck {
		t.Errorf("step3 (with check) body = %q, want %q", withCheck[2].body, wantStep3WithCheck)
	}
	if withCheck[2].actionLabel != "run relio check now" {
		t.Errorf("step3 actionLabel = %q, want %q", withCheck[2].actionLabel, "run relio check now")
	}

	// Step 4.
	withStatus := buildSteps(Context{RunStatus: func() error { return nil }})
	if withStatus[3].title != "See what's pending" {
		t.Errorf("step4 title = %q", withStatus[3].title)
	}
	if withStatus[3].body != "`relio status` lists the unreleased commits and the version they suggest." {
		t.Errorf("step4 body = %q", withStatus[3].body)
	}
	if withStatus[3].actionLabel != "run relio status now" {
		t.Errorf("step4 actionLabel = %q, want %q", withStatus[3].actionLabel, "run relio status now")
	}

	base := buildSteps(Context{})

	// Step 5.
	wantStep5 := "Run `relio` with no arguments: you get a preview, then a small wizard " +
		"(Create / change the bump / cancel). On confirm it writes the CHANGELOG.md section, " +
		"commits it as `chore(release): vX.Y.Z`, and creates the annotated tag — nothing before the confirm.\n" +
		"`relio --rc` cuts a release candidate you can iterate on; running `relio` again on an rc finalizes it."
	if base[4].title != "Create the release" {
		t.Errorf("step5 title = %q", base[4].title)
	}
	if base[4].body != wantStep5 {
		t.Errorf("step5 body = %q, want %q", base[4].body, wantStep5)
	}

	// Step 6 — with and without the no-token hint.
	wantStep6 := "Push with `git push --follow-tags`. Or `relio --publish` to push and create the GitHub Release " +
		"with the changelog notes as its body — that needs a GitHub token (GITHUB_TOKEN / GH_TOKEN / `gh auth login`)."
	if base[5].title != "Get it out" {
		t.Errorf("step6 title = %q", base[5].title)
	}
	if base[5].body != wantStep6+"\nNo GitHub token is set yet." {
		t.Errorf("step6 (no token) body = %q, want %q", base[5].body, wantStep6+"\nNo GitHub token is set yet.")
	}
	withToken := buildSteps(Context{HasToken: true})
	if withToken[5].body != wantStep6 {
		t.Errorf("step6 (with token) body = %q, want %q", withToken[5].body, wantStep6)
	}

	// Step 7.
	wantStep7 := "`relio post` prints announcement text for socials. `relio image` renders a PNG release card. " +
		"`.release.yaml` `version_files:` writes the new version into package.json / pyproject.toml / …. " +
		"`.release.yaml` `release.hooks.before` / `.after` run shell commands around the release."
	if base[6].title != "Optional extras" {
		t.Errorf("step7 title = %q", base[6].title)
	}
	if base[6].body != wantStep7 {
		t.Errorf("step7 body = %q, want %q", base[6].body, wantStep7)
	}

	// Step 8.
	wantHappyPath := "relio init → write feat:/fix: commits → relio status → relio → git push --follow-tags"
	wantStep8 := "Happy path:  " + wantHappyPath + ".\n" +
		"See `relio help` for every command and flag, and the README for the full `.release.yaml` reference."
	if base[7].title != "You're set" {
		t.Errorf("step8 title = %q", base[7].title)
	}
	if base[7].body != wantStep8 {
		t.Errorf("step8 body = %q, want %q", base[7].body, wantStep8)
	}

	// PrintPlain's own header and footer, byte-identical to the historical
	// hardcoded strings.
	var buf bytes.Buffer
	if err := PrintPlain(&buf, Context{HasRepo: true}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.HasPrefix(out, "Relio — guide\n") {
		t.Errorf("PrintPlain does not start with the expected header:\n%s", out)
	}
	wantFooter := "The happy path:  " + wantHappyPath
	if !strings.Contains(out, wantFooter) {
		t.Errorf("PrintPlain missing the expected footer %q:\n%s", wantFooter, out)
	}

	// Interactive stepper chrome: step counter and both footer variants,
	// byte-identical to the historical hardcoded strings.
	m := teaModel{steps: buildSteps(Context{RunInit: func() error { return nil }})}
	m.i = 1 // step 2 ("Set up") is the one carrying the RunInit action
	v := m.View()
	if !strings.Contains(v, "Step 2 of 8") {
		t.Errorf("View() missing step counter %q:\n%s", "Step 2 of 8", v)
	}
	if !strings.Contains(v, "[y] run relio init now · enter skip · ← back · q quit") {
		t.Errorf("View() missing the action footer:\n%s", v)
	}
	m2 := teaModel{steps: buildSteps(Context{})}
	m2.i = 2 // step 3 has no action in the base context
	v2 := m2.View()
	if !strings.Contains(v2, "enter continue · ← back · q quit") {
		t.Errorf("View() missing the no-action footer:\n%s", v2)
	}
}

// TestStepsLocalizeUnderSpanish proves guide.go's chrome and all eight steps
// actually route through i18n.T (not just accidentally identical in
// English): switching to "es" must change every one of them and must NOT
// leave the English literal behind.
func TestStepsLocalizeUnderSpanish(t *testing.T) {
	prev := i18n.Current()
	i18n.SetLanguage("es")
	t.Cleanup(func() { i18n.SetLanguage(prev) })

	ctx := Context{HasRepo: true, RunInit: func() error { return nil }, RunCheck: func() error { return nil }, RunStatus: func() error { return nil }}
	steps := buildSteps(ctx)

	wantTitles := []string{
		"Qué hace Relio",
		"Configura `.release.yaml`",
		"Escribe Conventional Commits",
		"Ve qué queda pendiente",
		"Crea el lanzamiento",
		"Publícalo",
		"Extras opcionales",
		"Ya estás listo",
	}
	for i, want := range wantTitles {
		if steps[i].title != want {
			t.Errorf("step %d title = %q, want %q", i+1, steps[i].title, want)
		}
	}

	for i, english := range []string{
		"What Relio does", "Set up", "Write Conventional Commits", "See what's pending",
		"Create the release", "Get it out", "Optional extras", "You're set",
	} {
		if strings.Contains(steps[i].body, english) || steps[i].title == english {
			t.Errorf("step %d still contains an English literal: %q", i+1, english)
		}
	}

	if steps[1].actionLabel != "ejecutar relio init ahora" {
		t.Errorf("step2 actionLabel = %q, want localized", steps[1].actionLabel)
	}
	if steps[2].actionLabel != "ejecutar relio check ahora" {
		t.Errorf("step3 actionLabel = %q, want localized", steps[2].actionLabel)
	}
	if steps[3].actionLabel != "ejecutar relio status ahora" {
		t.Errorf("step4 actionLabel = %q, want localized", steps[3].actionLabel)
	}

	var buf bytes.Buffer
	if err := PrintPlain(&buf, Context{HasRepo: true}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.HasPrefix(out, "Relio — guía\n") {
		t.Errorf("es PrintPlain does not start with the localized header:\n%s", out)
	}
	if !strings.Contains(out, "El camino feliz:  relio init → escribe commits feat:/fix: → relio status → relio → git push --follow-tags") {
		t.Errorf("es PrintPlain missing the localized footer:\n%s", out)
	}
	if strings.Contains(out, "The happy path") {
		t.Errorf("es PrintPlain still contains the English footer prefix:\n%s", out)
	}

	m := teaModel{steps: buildSteps(Context{RunInit: func() error { return nil }})}
	m.i = 1 // step 2 ("Set up") is the one carrying the RunInit action
	v := m.View()
	if !strings.Contains(v, "Paso 2 de 8") {
		t.Errorf("es View() missing localized step counter:\n%s", v)
	}
	if !strings.Contains(v, "[y] ejecutar relio init ahora · enter saltar · ← atrás · q salir") {
		t.Errorf("es View() missing the localized action footer:\n%s", v)
	}
	m2 := teaModel{steps: buildSteps(Context{})}
	m2.i = 2
	v2 := m2.View()
	if !strings.Contains(v2, "enter continuar · ← atrás · q salir") {
		t.Errorf("es View() missing the localized no-action footer:\n%s", v2)
	}
}
