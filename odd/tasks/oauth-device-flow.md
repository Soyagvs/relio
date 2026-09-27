# OAuth Device Flow login for `relio auth`

## Objective
Replace today's manual-token-only auth (`RELIO_GITHUB_TOKEN` / `GITHUB_TOKEN` / `GH_TOKEN` / `gh auth token`) with a real `relio auth login` that runs GitHub's OAuth Device Flow, so users no longer need to create a Personal Access Token by hand for `--publish`.

## Problem / why
README roadmap ("Next"): per-user GitHub sign-in via OAuth Device Flow with OS-keychain storage. Today `cmd/auth.go`'s `login`/`logout` are stub commands that just print static text (`AuthLoginBody`/`AuthLogoutBody`); there is no real auth flow.

## Scope
- GitHub OAuth App already registered by the user (Device Flow enabled, "Expire user authorization tokens" off). Client ID: `Ov23liuEAG3Q8f19pkUJ` (public, will be hardcoded — no client secret needed for Device Flow).
- New device-flow HTTP client (request device code, poll for access token).
- New hybrid token storage: OS keychain first (`zalando/go-keyring` — macOS Keychain / Windows Credential Manager / Linux Secret Service), falling back to a protected local file (0600, atomic write, same pattern as `internal/userconfig`) when no keychain backend is available (confirmed relevant on the user's WSL2 dev machine).
- Wire into `relio auth login` / `relio auth logout` / `relio auth status`, and into `ghrelease.Token()`'s existing priority chain (env vars still win for CI/script overrides; stored device-flow token sits before the `gh auth token` shell-out fallback).
- i18n strings (English + Spanish) for the new interactive flow.
- Docs: `docs/publishing.md`, `docs/commands.md`, README roadmap (move item from "Next" to "Shipped").

## Constraints
- Client ID is public and hardcoded — no secret to protect.
- `docs/publishing.md` currently states the token needs `repo` scope — device-flow request must ask for the same scope.
- Existing `APIBase` var in `internal/ghrelease` points at `api.github.com`; device flow endpoints live on `github.com` (different host) — needs its own overridable base var for test injection (`httptest.Server`), same idiom as `APIBase`.
- No spinner/progress UI primitive exists yet in `internal/ui` — polling UI is new code, kept minimal (plain text: show code + URL, print waiting dots, no bubbletea model required unless a task discovers it's cleaner).
- `cmd/auth_test.go` has a golden-string test (`TestGoldenEnglishAuthCmdUnchanged`) that will need updating once login/logout stop being static notes.
- CI: `go vet`, `gofmt`, `go test ./... -race -cover`, `golangci-lint` (new-code-only), `govulncheck`.

## TDD mode
**Strict TDD — enabled** (explicit session setting). Source: user/session config. Test runner: `go test ./...` (`make test` target). RED (failing test observed) → GREEN (minimal implementation) → REFACTOR, per task, before considering it done.

## Tasks

- [x] **T1 — Device flow HTTP client** (`internal/ghrelease`) — done
  - `internal/ghrelease/deviceflow.go` (`DeviceFlowBase`, `DeviceCode`, `RequestDeviceCode`, sentinel errors, `PollForToken` with cancellable sleep + slow_down backoff) + `internal/ghrelease/deviceflow_test.go` (9 tests: device-code success/GitHub-error/HTTP-error, poll success immediate/after-pending, slow_down interval bump, expired/denied via `errors.Is`, ctx cancellation mid-sleep).
  - Route: delegated writer (general-purpose, strict TDD). RED observed (compile failure on missing symbols), GREEN observed (`go test ./internal/ghrelease/... -race -v` — 9 new + 9 pre-existing tests pass, no -race warnings), independently re-verified by the parent orchestrator (`go vet`, `gofmt -l`, `go build`, spot-check re-run all clean).
  - Commit: see below.

- [x] **T2 — Hybrid token storage** (new `internal/tokenstore` package) — done
  - `internal/tokenstore/tokenstore.go` + `tokenstore_test.go`: `Store`/`Load`/`Delete`, keychain-first via injectable `keyringBackend` (real impl wraps `go-keyring`), file fallback mirrors `userconfig`'s atomic 0600 write. Added `github.com/zalando/go-keyring` to `go.mod`.
  - Route: delegated writer (general-purpose, strict TDD). RED observed (compile failure, undefined symbols), GREEN observed (6/6 tests pass). Orchestrator re-verified independently, found 4 errcheck issues golangci-lint flagged on unchecked `os.Remove`/`tmp.Close` in the temp-file cleanup paths (writer's own `go vet`/`gofmt` don't run golangci-lint) — fixed inline (mechanical, `_ =` discard, matches repo's existing errcheck convention), re-ran lint clean, full `go test ./...` green.
  - Commit: see below.

- [x] **T3 — Wire `cmd/auth.go` + `ghrelease.Token()` + i18n** — done
  - `authLogin`/`authLogout` in `cmd/auth.go` run the real device-flow login/logout via package-var seams (`requestDeviceCode`, `pollForToken`, `authenticatedUser`, `storeToken`, `loadToken`, `deleteToken`) mirroring the existing `lookupToken` idiom. `ghrelease.Token()` priority: env vars → `tokenstore.Load()` (var `loadStoredToken`) → `gh auth token`. `ghrelease.RelioOAuthClientID` const added (public Client ID `Ov23liuEAG3Q8f19pkUJ`).
  - New i18n keys: `AuthLoginInstruction`, `AuthLoginWaiting`, `AuthLoginSuccess`, `AuthLoginExpired`, `AuthLoginDenied`, `AuthLogoutSuccess`, `AuthLogoutNothingStored` (en+es). Removed dead `AuthLoginBody`/`AuthLogoutBody` (only referenced by the old stub commands). Repurposed `AuthLoginShort`/`AuthLogoutShort`/`AuthLong` text to describe the real flow instead of "how to set a token yourself".
  - `internal/menu` Auth entry untouched — it already just calls `authStatus`, no picker changes needed.
  - Route: delegated writer (general-purpose, strict TDD, 7 files). RED observed at each step (missing seam symbols; old golden test made a real network call and hung, proving the old stub path was gone). GREEN: all new + existing tests pass. Orchestrator re-verified independently: `go build`, `go vet`, `gofmt -l .`, targeted `go test -race` all clean; `golangci-lint` on touched packages shows 8 issues, confirmed via `git stash` diff to be byte-identical to the pre-T3 state (all in T1's `deviceflow.go`/`deviceflow_test.go` and untouched `ghrelease.go`/`cmd/init.go`) — zero new issues from T3.
  - Commit: see below.

- [x] **T4 — Docs** — done
  - `docs/publishing.md`: auth line now lists the stored device-flow token in the priority chain instead of claiming "no OAuth app, no browser flow"; example now leads with `relio auth login`.
  - `docs/commands.md`: `relio auth` section rewritten — real login/logout flow, expired/declined behavior, hybrid keychain/file storage note, updated status example.
  - `README.md`: roadmap line moved from "Next" to "Shipped"; "Next" section removed (nothing left there), "Later" (plugins, richer `post` templates) unchanged.
  - Route: direct inline (docs-only, mechanical, no design decision left once T1-T3 landed).
  - Commit: see below.

## Acceptance criteria
- `relio auth login` completes a real device-flow login against GitHub's real endpoints (manually verified once by the user, since no live GitHub server in CI) and stores a working token.
- `relio auth status` reports the stored token's source distinctly from env-var/`gh` sources.
- `relio auth logout` clears the stored token and `status` reflects that (falls through to next source or reports not authenticated).
- `go test ./...`, `go vet ./...`, `gofmt -l .` clean; new code covered by tests (keyring path forced to fallback in tests, file path asserted directly, device-flow polling tested against `httptest.Server` fixtures for pending/slow_down/success/expired/denied).
- Existing token-based `--publish` flow keeps working unchanged (env var / `gh auth token` priority preserved).

## Progress
- Branch: `feat/oauth-device-flow` (created from `main`).
- 2026 session: GitHub OAuth App created by user, Device Flow enabled, token expiry disabled, wildcard callback matching disabled. Client ID captured above.
- Codebase mapped (token priority chain, userconfig storage shape, no existing keychain dep, menu wiring pattern, i18n key pattern, TDD/CI commands) — see task notes above for the concrete findings.
- Storage decision made by user: **hybrid** (keychain-first, file-fallback) — chosen over pure-keychain (would break on user's WSL2 box without gnome-keyring/kwallet) and over file-only (wouldn't honor the existing "OS keychain" doc promise).

## Follow-up fix (post-T1-T4)
Gentle AI's review of the T1+T2 candidate flagged (WARNING, non-blocking): `tokenstore.Delete()` discarded every keychain error unconditionally, so a genuine deletion failure (not just "unavailable"/"not found") would be silently treated as success, leaving `relio auth logout` falsely reporting success while the token stayed live in the keychain. Fixed: `Delete()` now verifies the outcome with a follow-up `Get` instead of trusting the keychain's own error — mirrors `Store`'s existing "don't trust keychain error codes, WSL2 errors on everything" reasoning. New test `TestDeleteReturnsErrorWhenKeychainDeletionActuallyFails` (plus a `deleteErr` knob on the test fake to simulate a real deletion failure distinct from "backend unavailable"). Direct inline fix (single file, mechanical once diagnosed), TDD: RED confirmed, GREEN confirmed, full `go test ./...` + lint clean.

## Known limitations (from the final 4-lens high-risk review, all approved/non-blocking)
None of these opened a correction; all are informational follow-ups, not defects in what was asked for. Grouped by how worth doing they are:

**Quick, worth a future pass:**
- `PollForToken`'s initial `interval` (from `DeviceCode.Interval`, GitHub-supplied) is used unclamped — a zero/negative value would busy-loop against GitHub's token endpoint (readability R2-1, reliability R3-expires-in-boundary).
- `authLogin`'s `RequestDeviceCode` call runs under bare `context.Background()` with no deadline, unlike `authStatus`'s existing `10*time.Second` timeout two lines above it — inconsistent, and a stalled GitHub connection hangs `relio auth login` forever (resilience R4-authlogin-no-deadline).
- `i18n.AuthNotAuthenticated` still only mentions `GITHUB_TOKEN`/`gh auth login`, not the new `relio auth login` — stale text now that this candidate rewrote every other auth string (readability R2-3).
- `ErrAuthorizationPending`/`ErrSlowDown` are exported but documented as "never returned to the caller" — either unexport them or fix the comment (readability R2-2).

**Bigger, real, but out of this feature's scope:**
- `PollForToken` aborts the entire login on any transient network error/non-2xx instead of retrying — a single Wi-Fi blip during the up-to-15-minute polling window forces a full restart (resilience R4-pollfortoken-no-retry).
- Neither `tokenstore`'s `keyringBackend` calls nor `ghrelease.Token()`'s call into it carry a context/timeout — a hung OS keychain prompt (e.g. an unanswerable Secret Service unlock in a headless session) blocks indefinitely (resilience R4-tokenstore-keychain-hang, reliability R3-keychain-no-timeout).
- `Delete()`'s verification-read can itself transiently fail, in which case a genuine deletion failure still reads as success (risk R1-001) — a residual edge case on top of the fix already applied this session.
- File-fallback token storage is inherently cleartext-on-disk (risk R1-002) — this is the accepted tradeoff from the hybrid-storage decision, not a bug, but worth knowing it's there.
- A few defensive branches in `pollOnce` (malformed/non-2xx poll responses) and one timing-sensitive test (`TestPollForTokenSlowDownIncreasesInterval`) remain untested (reliability R3-polloncecoverage, R3-slowdown-timing-flake).

## Next step
All four tasks (T1-T4) plus the post-review Delete() fix are implemented, tested, and committed on `feat/oauth-device-flow`, with three review cycles (T1; T1+T2; T1-T4+fix) all approved. Remaining before this can be considered fully closed:
- **Manual live verification**: run `relio auth login` for real against GitHub (no CI/test can do this — it needs a live browser approval). Confirm the token lands in the keychain (or file fallback, if testing on WSL2) and that `relio auth status` / `--publish` pick it up.
- Decide whether to act on any "known limitations" above now or later.
- Push the branch and open a PR when the user is ready (not done automatically — delivery stays the user's decision).

## Engram mirror status
**Pending** — `mem_save` to topic `odd/oauth-device-flow/tasks` failed both attempts with `multiple active runtime sessions match the current project and directory` (likely another Claude Code session open on this repo). This file remains the source of truth until the mirror succeeds; retry later, do not block implementation on it.
