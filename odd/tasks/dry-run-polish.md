# Dry-run polish + small TUI i18n fixes

## Objective
Close out the non-blocking findings left by the `--dry-run` feature's final RDD review, plus one leftover i18n gap from the earlier auth-menu fix. Small, bounded batch — user said "empieza por lo chico, luego por lo grande" (start small, then the bigger TUI gaps: `releases list/delete`, self-update, upload retry — tracked separately, not in this batch).

## Scope
1. **Shared tag-collision check** (`internal/release/release.go`): extract `Plan.Apply`'s pre-write tag-collision guard into a reusable `Plan.CheckTagCollision(repo *gitrepo.Repo) error` method with a new `ErrTagExists` sentinel, so `cmd/root.go`'s `dryRunPreview` (added by the dry-run feature) stops duplicating that logic — the exact risk the readability lens flagged (`R2-002`): the two checks could silently drift.
   - `Apply` keeps calling it as its own first step (same behavior, same error text shape so `TestApplyRefusesDuplicateTag`'s `strings.Contains(err.Error(), "already exists")` still passes).
   - `dryRunPreview` calls it directly, and on `errors.Is(err, release.ErrTagExists)` wraps it into a properly i18n'd message (`DryRunTagAlreadyExists`) instead of today's raw `fmt.Errorf("tag %s already exists", name)` — closes `R2-001`/`R3-002`/`R3-3` (three lenses independently flagged the same untranslated string).
   - A `repo.HasTag` failure (not a collision, a real error — corrupted repo, I/O) gets wrapped with context via a new i18n key (`DryRunTagCheckFailed`) instead of today's bare `return err` — closes `R4-hastag-unwrapped`.
2. **Fail-fast ordering** (`cmd/root.go` `dryRunPreview`): run the (now-shared) collision check *before* printing the full changelog text, not after — closes `R4-changelog-before-failfast`/`R3-2`: today a colliding dry-run still prints the whole changelog body before erroring.
3. **Test the untested branches** (`cmd/root_test.go`): `dryRunPreview`'s `ChangelogSkipped` (`--no-changelog`), `NoTag` (`--no-tag`), and `WouldPublish`/`WouldNotPublish` (`--publish`) output lines have no test asserting their actual text — closes `R3-1`/`R4-untested-degraded-branches`.
4. **Menu i18n gap** (`cmd/root.go` `runMenuSetup`): the "not a git repository" line is still hardcoded English (`ui.Info("not a git repository — run this inside a repo, or pass -C <path>")`) — same class of bug as the auth-menu fix earlier this session. New key `MenuSetupNotAGitRepo` (en+es). NOT the same key as `i18n.InitNotAGitRepo` (`cmd/init.go`) — different wording/context (that one says "run `relio init`", this one is about `-C <path>`).
5. **Pre-existing lint cleanup** (`cmd/init.go:28`): `golangci-lint`'s `SA1006` (printf-style function with dynamic format string and no further arguments) has shown up as a "pre-existing, unrelated" finding in every review this session (`cmd/init.go:28: fmt.Errorf(i18n.T(i18n.InitNotAGitRepo))`). Since we're touching i18n/error-handling in this same area anyway, fix it: `errors.New(i18n.T(i18n.InitNotAGitRepo))` instead of `fmt.Errorf` with no format args.

## Constraints
- `Plan.CheckTagCollision` must return an error whose text still contains `"already exists"` for `TestApplyRefusesDuplicateTag`'s existing assertion (`internal/release/release_test.go:762`) to keep passing unchanged.
- `internal/release` does not import `internal/i18n` (presentation-layer concern stays in `cmd/`) — `ErrTagExists` is a plain sentinel; the cmd package translates it.
- `TestDryRunPreviewReportsExistingTagCollision` (`cmd/root_test.go`) already asserts `strings.Contains(err.Error(), "tag v1.6.0 already exists")` — keep the EN catalog text for `DryRunTagAlreadyExists` semantically identical (`"tag %s already exists"`) so this test doesn't need rewriting, just confirm it still passes.
- i18n templates in this codebase do carry literal `%w` for `fmt.Errorf(i18n.T(key), args..., err)` — see `i18n.PublishNoOriginRemote` for the existing precedent; follow the same shape for `DryRunTagCheckFailed`.
- Don't touch `releases list/delete`, self-update, or upload retry — those are the "grande" (big) items, explicitly deferred to a later round.

## TDD mode
**Strict TDD — enabled** (session setting, same as prior features this session). Test runner: `go test ./...`. RED → GREEN → REFACTOR.

## Tasks

- [x] **T1 — Shared collision check + i18n'd dry-run errors + fail-fast ordering + untested branches + menu i18n + init.go lint fix** — done
  - Single cohesive task (all five items touch the same small cluster of files and are cheap enough not to warrant separate task-file entries).
  - **Shared collision check** (`internal/release/release.go`): new `var ErrTagExists = errors.New("release: tag already exists")` sentinel and `Plan.CheckTagCollision(repo *gitrepo.Repo) error` — a no-op returning nil when `TagUpdate` is false, otherwise `repo.HasTag(name)` and, on a hit, `fmt.Errorf("tag %s already exists: %w", name, ErrTagExists)` (still contains `"already exists"`, now also matchable via `errors.Is`). `Apply` calls it as its own first step (`if err := p.CheckTagCollision(repo); err != nil { return res, err }`) instead of the old inline `if p.TagUpdate { ... }` block — same observable behavior, same error text shape.
  - **i18n'd dry-run errors** (`cmd/root.go` `dryRunPreview`): dropped the duplicated inline `repo.HasTag`/`fmt.Errorf` block; now calls `plan.CheckTagCollision(repo)` once. `errors.Is(err, release.ErrTagExists)` → `fmt.Errorf(i18n.T(i18n.DryRunTagAlreadyExists), plan.TagName())` (EN `"tag %s already exists"`, ES `"la etiqueta %s ya existe"`). Any other (non-sentinel) error → `fmt.Errorf(i18n.T(i18n.DryRunTagCheckFailed), plan.TagName(), err)` (EN `"checking whether tag %s exists: %w"`, ES `"comprobando si la etiqueta %s existe: %w"`, following the `PublishNoOriginRemote` `%w`-in-template precedent).
  - **Fail-fast ordering**: `CheckTagCollision` now runs as the very first statement in `dryRunPreview`, before the `DryRunHeader` line, the changelog text, or anything else is printed — a colliding dry-run now errors with zero output instead of printing the full changelog body first.
  - **Untested branches** (`cmd/root_test.go`): added `TestDoReleaseDryRunNoChangelogReportsSkipped` (`--no-changelog`, asserts the `"changelog would not be updated (--no-changelog)"` line), `TestDoReleaseDryRunNoTagReportsNoTag` (`--no-tag`, asserts `"no tag would be created (--no-tag)"`), `TestDoReleaseDryRunPublishReportsWouldPublish` (`--publish`, asserts `"would push and publish the GitHub Release"`) — all three via `repoWithPendingRelease` + `doRelease(&buf, r, config.Default("proj"), f, semver.None, false)` with `dryRun: true`, matching the existing dry-run test pattern.
  - **Menu i18n gap** (`cmd/root.go` `runMenuSetup`): `ui.Info("not a git repository — run this inside a repo, or pass -C <path>")` → `ui.Info(i18n.T(i18n.MenuSetupNotAGitRepo))`. New key, EN text identical to the old hardcoded string, ES `"no es un repositorio git — ejecútalo dentro de un repo, o pasa -C <ruta>"`. Deliberately a separate key from `i18n.InitNotAGitRepo` (different wording/context, per the task brief).
  - **`cmd/init.go:28` lint fix**: `return fmt.Errorf(i18n.T(i18n.InitNotAGitRepo))` → `return errors.New(i18n.T(i18n.InitNotAGitRepo))`; added the `"errors"` import (wasn't present).
  - **RED observed** (each confirmed failing before its fix, in order):
    - `internal/release/release_test.go` — added `TestCheckTagCollisionDetectsExistingTag`, `TestCheckTagCollisionNoCollisionReturnsNil`, `TestCheckTagCollisionSkipsWhenTagUpdateDisabled` (plus `errors` import): `go test ./internal/release/...` failed to compile — `p.CheckTagCollision undefined (type Plan has no field or method CheckTagCollision)` / `undefined: ErrTagExists` (3 call sites).
    - `cmd/root_test.go` — added `TestDryRunPreviewWrapsTagCheckFailure` (breaks the repo via `os.RemoveAll(filepath.Join(dir, ".git"))` after `BuildPlan`, so the shared check's `git tag --list` fails for a non-collision reason): failed with `error = "git tag --list v1.6.0: fatal: not a git repository (or any of the parent directories): .git", want context about checking tag v1.6.0` — proved the old bare `return err` gave no context.
    - `TestDoReleaseDryRunNoChangelogReportsSkipped`/`NoTagReportsNoTag`/`PublishReportsWouldPublish` and `cmd/menu_test.go`'s `TestRunMenuSetupNotAGitRepo` passed immediately on first run — expected, not a TDD gap: `dryRunPreview`'s three branches and `runMenuSetup`'s error line already worked correctly pre-change, they just had no test asserting the text yet (same as the dry-run-flag task's precedent for pre-existing-but-untested branches).
  - **GREEN observed**: after implementing `CheckTagCollision`/`ErrTagExists` + the `Apply` refactor, the 3 new `internal/release` tests passed and `TestApplyRefusesDuplicateTag` kept passing unchanged. After the `dryRunPreview` refactor (shared check, i18n, fail-fast reordering), `TestDryRunPreviewWrapsTagCheckFailure` passed, `TestDryRunPreviewReportsExistingTagCollision` kept passing unchanged (still asserts `strings.Contains(err.Error(), "tag v1.6.0 already exists")`), and the three new branch-coverage tests plus every pre-existing `cmd` dry-run test stayed green. After the `runMenuSetup`/`init.go` swaps, `TestRunMenuSetupNotAGitRepo` and `TestRunMenuSetupOnExistingConfig` both stayed green.
  - **Verification** (this session, actual output):
    - `go build ./...` — clean, no output.
    - `go vet ./...` — clean, no output.
    - `gofmt -l .` — empty (one intermediate run flagged `internal/i18n/keys.go` for const-block alignment after the new `DryRunTagAlreadyExists`/`DryRunTagCheckFailed`/`MenuSetupNotAGitRepo` keys were added; `gofmt -w internal/i18n/keys.go` fixed it, re-run confirmed empty).
    - `go test ./cmd/... ./internal/release/... ./internal/i18n/... -race -v` — all pass, including `TestCatalogKeyParity`, `TestCatalogVerbArityParity`, `TestKeysDeclaredInASTExistInEnglishCatalog` (en/es parity + `%w`/`%s` verb-arity parity for the 3 new keys), `TestApplyRefusesDuplicateTag`, all `TestCheckTagCollision*`, all `TestDoReleaseDryRun*`/`TestDryRunPreview*`, `TestRunMenuSetupNotAGitRepo`, `TestRunMenuSetupOnExistingConfig`, and every other pre-existing test in the three packages.
    - `go test ./... -race` — all 24 packages pass (2 report "no test files": the root `cmd`-consuming `main` package and `tools/statsnap`).
    - `golangci-lint run ./...` — 30 issues (errcheck 24, staticcheck 5, unused 1), all pre-existing and unrelated (none in `cmd/init.go`, `cmd/root.go`, `internal/release/release.go`, or `internal/i18n/*`); confirmed via `git stash`/`git stash pop` that the pre-change baseline had 31 issues (staticcheck 6, including `cmd/init.go:28: SA1006: printf-style function with dynamic format string and no further arguments should use print-style function instead`) — the fix removed exactly that one finding and introduced zero new ones.
  - **Judgment calls for review**:
    1. `ErrTagExists`'s own sentinel message (`"release: tag already exists"`) is never surfaced directly — `CheckTagCollision` always wraps it into `"tag %s already exists: %w"` before returning, so the sentinel text itself is only reachable via `errors.Unwrap`/`%v` formatting of the wrapped chain, matching how the task described it ("plain sentinel error"; the cmd package does all user-facing translation).
    2. Added a direct `internal/release` unit-test trio for `CheckTagCollision` beyond the "if worth adding" hint in the task brief, since it's a new exported method with its own branch (`TagUpdate` false is a no-op) that `TestApplyRefusesDuplicateTag` alone doesn't exercise.
    3. Added `TestDryRunPreviewWrapsTagCheckFailure` (not explicitly requested) to cover the non-sentinel-error wrap path (`DryRunTagCheckFailed`) — simulates a real `git` failure by deleting `.git` after `BuildPlan`, no new seam invented.
    4. Added `TestRunMenuSetupNotAGitRepo` (not explicitly requested) since no existing test exercised that branch of `runMenuSetup` at all; text-only assertion, no behavior change from the user's point of view (same EN wording, now i18n-routed).
    5. `DryRunTagAlreadyExists`'s EN template omits `%w` (unlike `DryRunTagCheckFailed`) — it's a fresh user-facing message, not a wrap of a technical error, so there's nothing to preserve via `%w`; only the already-known tag name is interpolated.
  - Files changed: `internal/release/release.go`, `internal/release/release_test.go`, `cmd/root.go`, `cmd/root_test.go`, `cmd/menu_test.go`, `cmd/init.go`, `internal/i18n/keys.go`, `internal/i18n/catalog_en.go`, `internal/i18n/catalog_es.go`.
  - Route: delegated writer (this session, strict TDD) — not committed; left uncommitted per the task's instructions for the orchestrator's diff review + RDD.

## Acceptance criteria
- `TestApplyRefusesDuplicateTag`, `TestDryRunPreviewReportsExistingTagCollision`, and every other existing test keep passing unchanged (or with only mechanical updates if wording genuinely had to change — flag any such case).
- A colliding `--dry-run` no longer prints the changelog text before erroring.
- The tag-collision error and the `HasTag`-failure error inside `dryRunPreview` are both i18n'd (en+es).
- `--dry-run --no-changelog`, `--dry-run --no-tag`, and `--dry-run --publish` each have a test asserting their specific preview output text.
- The menu's "not a git repository" line is i18n'd (en+es), matching the auth-menu fix's pattern from earlier this session.
- `golangci-lint`'s `cmd/init.go:28` SA1006 finding is gone.
- `go build`, `go vet`, `gofmt -l .`, `go test ./... -race`, `golangci-lint run ./...` all clean.

## Progress
- Committing directly to `main` per this session's established pattern (user preference, confirmed: PRs only for repos that aren't their own).
- RDD (receipt-driven development) is on for this repo — expect the same consent/4-lens review cycle as the prior features this session before this can be reported done.
- T1 implemented and fully verified this session (see task entry above for RED/GREEN evidence and exact verification output); left uncommitted for the orchestrator's diff review, per this task's own instructions.

## Next step
T1's diff is ready for the orchestrator to review and commit. Remaining after that:
- Run through RDD review (consent/4-lens cycle) before this is reported fully done.
- Push when the user is ready (not automatic — delivery stays the user's decision).
