# `--dry-run` flag for `relio` release

## Objective
Add a `--dry-run` flag that runs the full release plan (version bump, changelog text, version-file diffs, hook validation) and prints what *would* happen, without any side effect: no changelog file write, no version-file write, no git commit, no git tag, no push, no GitHub Release creation, no `$EDITOR` open, no Before/After hooks.

## Problem / why
User-driven TUI gap analysis flagged this as the top missing safety net: `relio` always commits/tags for real once past the wizard/`--yes` gate — there is no way to preview a release without touching the repo. Recommended by the assistant, accepted by the user (2026-09-28 session).

## Scope
- New `releaseFlags.dryRun` bool + `--dry-run` cobra flag on the root command.
- `doRelease` (cmd/root.go:522) gets a non-mutating path: when `f.dryRun`, skip `plan.Apply(repo)` entirely and instead call the same non-mutating building blocks Apply would use (changelog text generation, `versionfile.Plan`-style diff) directly, print them, and stop before any git/GitHub call.
- i18n keys for the new flag's usage string + preview labels (en+es).
- Docs: README, docs/commands.md, docs/publishing.md.

## Constraints (from mapping — see `codegraph_explore` findings this session)
- `plan.Apply(repo)` (internal/release/release.go:328) does, in order: tag-collision check (read-only) → write changelog file (os.WriteFile) → write version files (versionfile.Apply) → git commit (repo.CommitPaths) → git tag (repo.CreateTag). Dry-run must not call `Apply` at all.
- `versionfile` already has a non-mutating step separate from `Apply` (returns `Change{Old,New}` per file) — reuse it for the diff preview instead of inventing new diffing.
- Changelog text generation is a separate step from the file write inside Apply — call that generator directly to get the text to print, without writing.
- Hooks: `Validate`/`Before`/`After` all use the identical shell-out mechanism (`internal/hook/hook.go:25`) — there is no read-only guarantee at the type level. Decision (user, 2026-09-28): **Validate runs under dry-run** (it's the pre-flight check, gives real go/no-go signal); **Before/After do NOT run** under dry-run (they're modeled as real actions). `--no-hooks` continues to gate all three independently of `--dry-run`.
- Changelog preview depth (user, 2026-09-28): **print the full generated changelog text**, not just the plan summary — buffer it, don't write it.
- `publishGitHubRelease` (cmd/publish.go:29) already requires a real token and does real `repo.Push` + `ghrelease.Create` — dry-run just needs to never call it (no internal no-op path needed inside that function).
- `--edit` ($EDITOR) should be skipped under dry-run — show the generated notes, don't open an editor.
- `ui.PlanView` (internal/ui/ui.go:575) is a plain `func(release.Plan) string`, not TUI-coupled, already used in both interactive and non-interactive paths before the wizard/`--yes` branch — reuse for the top-of-preview summary.
- Applies uniformly across all `doRelease` call sites (direct non-interactive, interactive wizard-confirmed, and the menu's `runMenuRelease` 3-step picker) since they all funnel through the same `releaseFlags` struct and the same `doRelease` call — no separate menu picker entry needed for v1.
- Existing test seam: `repoWithPendingRelease(t)` (cmd/root_test.go:19) builds a real temp-dir git repo; existing hook-abort tests already assert "no tag created" via `r.HasTag(...)` — same pattern for asserting dry-run wrote nothing.

## TDD mode
**Strict TDD — enabled** (session setting, same as the oauth-device-flow feature). Test runner: `go test ./...`. RED → GREEN → REFACTOR per task.

## Tasks

- [x] **T1 — `--dry-run` flag + non-mutating preview path in `doRelease`** — done
  - `releaseFlags.dryRun` field (`cmd/root.go:53`), cobra registration (`lf.BoolVar(&f.dryRun, "dry-run", false, i18n.T(i18n.FlagDryRunUsage))`, next to `--edit`), `FlagDryRunUsage` i18n key (en+es).
  - `doRelease`: inserted a single `if f.dryRun { return dryRunPreview(writeLine, plan) }` branch right after the interactive-wizard/non-interactive-`--yes` confirmation gate and before the `--edit` check — so Validate hooks (which run earlier, unchanged) still gate a dry-run, while `--edit`, Before hooks, `plan.Apply`, `publishGitHubRelease`, and After hooks are all unreached on this path. One code path covers every caller (direct `--yes`, wizard-confirmed, and the menu's `runMenuRelease`) since they all set fields on the same `*releaseFlags` and call the same `doRelease` — no special-casing needed.
  - New `dryRunPreview` helper (`cmd/root.go`, right before `footerTip`) reuses the plan's own non-mutating building blocks instead of duplicating logic: `plan.Section()` (the same changelog-text generator `Apply` feeds into `changelog.Update` before writing) for the full changelog text, `plan.TagName()` for the tag/version that would be created, and `plan.PublishGitHub` for the publish-intent line. `plan.VersionChanges` (the old→new version-file diff, already computed non-mutating by `BuildPlan` via `versionfile.Plan`) is not re-printed by `dryRunPreview` — it's already shown by the existing unconditional `ui.PlanView(plan)` call earlier in `doRelease`, on both the interactive and non-interactive paths, so a dry run's version-file-diff visibility was already correct before this task; duplicating it would have violated "reuse rather than invent."
  - New i18n keys (en+es, one key per printed line, matching `authLogin`'s convention): `FlagDryRunUsage`, `DryRunHeader` ("Dry run — nothing will be written."), `DryRunChangelogLabel`, `DryRunChangelogSkipped` (shown instead of the changelog text when `--no-changelog` zeroed `plan.ChangelogUpdate`), `DryRunTagWouldBeCreated` (`"tag %s would be created"`), `DryRunNoTag` (shown when `--no-tag` zeroed `plan.TagUpdate`), `DryRunWouldPublish`, `DryRunWouldNotPublish`.
  - Tests added to `cmd/root_test.go`: `TestRootHasDryRunFlag`/`TestRootDryRunFlagWiresToStruct` (flag exists and cobra parsing sets the bound value); `TestDoReleaseDryRunLeavesRepoUntouched` (`repoWithPendingRelease` + `f.dryRun=true`, asserts `doRelease` returns nil, `r.HasTag("v1.6.0")` is false, `CHANGELOG.md` was never written, `git log` has no `chore(release)` commit, and the output contains both the tag and the generated changelog text "Something big"); `TestDoReleaseDryRunValidateHookStillAborts` (Validate hook `exit 1` still fails a dry run and no tag is created — proves dry-run doesn't bypass validation); `TestDoReleaseDryRunSkipsBeforeAfterHooks` (Before/After hooks each `touch` a marker file in the repo dir; dry run succeeds and neither marker exists — no invocation-counting seam existed in the codebase to reuse, so a marker file was the simplest correct proof, matching the task brief's own suggested fallback); `TestDoReleaseDryRunSkipsEditor` (reuses `editor.Edit`'s existing `$RELIO_EDITOR` env-var seam — same one `internal/editor/editor_test.go` uses — set to `"false"`, which exits non-zero; if `--edit` were honored under dry-run, `doRelease` would surface `"editing release notes: ..."` and fail, so a nil return is the proof the editor was never invoked; no new seam invented).
  - RED observed before implementation: `TestDoReleaseDryRunLeavesRepoUntouched` failed with "dry-run created tag v1.6.0" / "dry-run wrote CHANGELOG.md" / a `chore(release)` commit in the log; `TestDoReleaseDryRunSkipsBeforeAfterHooks` failed with "the before hook ran" / "the after hook ran"; `TestDoReleaseDryRunSkipsEditor` failed with `doRelease` returning `"--edit needs an interactive terminal"` (the dry-run branch didn't exist yet, so `--edit`'s existing non-interactive guard fired first). `TestRootHasDryRunFlag`/`TestRootDryRunFlagWiresToStruct`/`TestDoReleaseDryRunValidateHookStillAborts` passed immediately since the flag/struct field/Validate-hook path needed no new dry-run branching — expected, not a TDD gap.
  - GREEN observed after adding the `f.dryRun` branch + `dryRunPreview`: all 6 new tests pass; full existing `cmd` package suite (hook-abort, `--edit`, `--rc`, footer-tip tests) unaffected.
  - Verification (this session, all actual output, not summarized):
    - `go build ./...` — clean.
    - `go vet ./...` — clean.
    - `gofmt -l .` — empty (one pre-edit run flagged `internal/i18n/keys.go` for const-block alignment after the new keys were added; `gofmt -w` fixed it, re-run confirmed empty).
    - `go test ./cmd/... ./internal/release/... ./internal/i18n/... -race -v` — all pass, including `TestCatalogKeyParity`, `TestCatalogVerbArityParity`, `TestKeysDeclaredInASTExistInEnglishCatalog` (en/es parity + fmt-verb arity for the 8 new keys) and every pre-existing hook/`--edit`/`--rc` test in `cmd`.
    - `go test ./... -race` — all packages pass.
    - `golangci-lint run ./cmd/... ./internal/release/... ./internal/i18n/...` — 1 finding (`cmd/init.go:28`, SA1006 on `fmt.Errorf(i18n.T(...))` with no args); confirmed via `git stash`/`git stash pop` to be byte-identical pre-existing on `main`, unrelated to `cmd/init.go` which T1 never touched — reported as a known pre-existing issue, not a T1 regression.
  - Judgment calls for review: (1) the "would" preview lines are gated on `plan.ChangelogUpdate`/`plan.TagUpdate`/`plan.PublishGitHub` rather than always printed unconditionally, so `--dry-run --no-tag`/`--no-changelog` report accurately instead of claiming a tag/changelog write that a real run wouldn't make either — this reads truer than the task brief's literal wording but wasn't asked for explicitly, flagging for confirmation it's wanted; (2) chose not to duplicate `ui.PlanView`'s version-file-diff output inside `dryRunPreview` since it's already unconditionally printed earlier in `doRelease` on every path — flagging in case the intent was for the dry-run block to be self-contained/standalone output instead of relying on the earlier print.
  - Files changed: `cmd/root.go`, `cmd/root_test.go`, `internal/i18n/keys.go`, `internal/i18n/catalog_en.go`, `internal/i18n/catalog_es.go`.
  - Route: delegated writer (this session, strict TDD) — not committed; left uncommitted for the orchestrator's diff review + RDD per the task's instructions.

## Follow-up fix (orchestrator review, same session, before commit)
While reviewing the T1 diff via `codegraph_explore` before committing, found the delegated writer had placed the `if f.dryRun { return dryRunPreview(...) }` check *after* the interactive wizard confirm / non-interactive `--yes` gate — meaning `relio --dry-run` alone (no `--yes`, no TTY confirm) still hit `"refusing to modify the repo without confirmation — re-run with --yes"` before ever reaching the dry-run branch, defeating the whole point of a confirmation-free preview. All 4 of T1's own new tests had set `yes: true` alongside `dryRun: true`, so this never surfaced.

Fixed directly (orchestrator, TDD): added `TestDoReleaseDryRunSkipsConfirmation` (`yes: false, dryRun: true`, asserts `doRelease` returns nil and the tag preview is printed) — RED confirmed (`"refusing to modify the repo without confirmation"`). Moved the `f.dryRun` check to immediately after the Validate-hook block and before the interactive/non-interactive branching, printing `ui.PlanView(plan)` itself there and returning before either confirmation path is reached; removed the old post-gate check. GREEN confirmed: new test passes, all pre-existing `cmd`/`internal/release`/`internal/i18n` tests still pass (`go test ./cmd/... ./internal/release/... ./internal/i18n/... -race -v`), full suite green (`go test ./... -race`), `go build`/`go vet`/`gofmt -l .` clean, `golangci-lint` shows only the same pre-existing `cmd/init.go` finding (confirmed unrelated). Files touched: `cmd/root.go`, `cmd/root_test.go`.

- [x] **T2 — Docs** — done
  - `docs/commands.md`: added `--dry-run` to the release-flags table (with the validate-runs/before-after-skip/no-yes-needed behavior spelled out) and two example invocations.
  - `docs/publishing.md`: new "Previewing before you publish" section pointing at `relio --dry-run --publish`.
  - `README.md`: added to the roadmap's "Shipped" line.
  - Route: direct inline (docs-only, mechanical, no design decision left once T1 landed).

## Acceptance criteria
- `relio --dry-run` (and `relio --dry-run --publish`) prints the full plan (version, tag, full changelog text, version-file diffs, publish intent) and exits 0 with the repo byte-for-byte unchanged (no new commit, no new tag, no changelog/version-file writes).
- Validate hooks still run and can abort a dry-run (proving whether the real release would pass validation).
- Before/After hooks and `$EDITOR` do not run under dry-run.
- `go test ./...`, `go vet ./...`, `gofmt -l .` clean.
- Existing non-dry-run release flow is unchanged.

## Progress
- Branch: none yet — committing directly to `main` per this session's established pattern (see prior oauth-device-flow-followup fix, commit `9714f53`, RDD-reviewed and approved on `main`).
- Mapping done this session via delegated `codegraph_explore`-based exploration (see file:line evidence in Constraints above).
- Two product decisions resolved via user AskUserQuestion (2026-09-28): Validate-only hooks under dry-run; full changelog text in preview.

## Next step
Delegate T1 to a writer (strict TDD), review its diff, then T2 docs.

## Engram mirror status
**Pending** — `mem_save` to topic `odd/dry-run-flag/tasks` failed: "multiple active runtime sessions match the current project and directory" (same recurring issue as the oauth-device-flow feature's mirror). This file remains the source of truth; retry later, not blocking implementation.
