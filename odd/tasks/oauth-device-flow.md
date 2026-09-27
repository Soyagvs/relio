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

- [ ] **T2 — Hybrid token storage** (new `internal/tokenstore` package)
  - `Store(token string) error`, `Load() (token, source string, err error)`, `Delete() error`.
  - Tries OS keychain (`go-keyring`, service `relio`, user `github`) first; on any keyring error, falls back to a protected file (reuse `userconfig`-style atomic write, 0600, under `os.UserConfigDir()/relio/`).
  - Add `github.com/zalando/go-keyring` to `go.mod`.
  - Keyring access goes through an interface so tests never touch a real OS keychain — force the fallback path in tests and assert file permissions/content.
  - Route: delegated writer (new package, 2 files — impl + test — nontrivial: two backends).

- [ ] **T3 — Wire `cmd/auth.go` + `ghrelease.Token()` + i18n**
  - `relio auth login`: `RequestDeviceCode` → print code + verification URL (i18n) → `PollForToken` → `tokenstore.Store` → confirm via `AuthenticatedUser`.
  - `relio auth logout`: `tokenstore.Delete`, confirm.
  - `relio auth status`: unchanged shape, but `ghrelease.Token()` priority becomes: env vars (unchanged) → `tokenstore.Load()` → `gh auth token` (unchanged last resort).
  - New i18n keys in `keys.go` + `catalog_en.go` + `catalog_es.go` for: instructions, waiting/polling message, success, denied, expired, keychain-unavailable-using-file note.
  - Update `cmd/auth_test.go` (golden string test + new behavior tests); update `internal/menu` Auth entry only if the picker needs a new "Login"/"Logout" choice (check during implementation — today menu's Auth entry just calls `authStatus`).
  - Route: delegated writer (touches cmd/auth.go, i18n x3, auth_test.go, ghrelease.go priority chain — 5+ files, clearly over the threshold).

- [ ] **T4 — Docs**
  - `docs/publishing.md`: replace "there is no OAuth app" with the real flow description.
  - `docs/commands.md`: document `relio auth login`/`logout` real behavior.
  - `README.md`: move the roadmap line from "Next" to "Shipped".
  - Route: direct inline (docs-only, mechanical, no design decision left once T1-T3 land).

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

## Next step
Start T1 (device flow HTTP client) with a delegated writer under strict TDD.

## Engram mirror status
**Pending** — `mem_save` to topic `odd/oauth-device-flow/tasks` failed both attempts with `multiple active runtime sessions match the current project and directory` (likely another Claude Code session open on this repo). This file remains the source of truth until the mirror succeeds; retry later, do not block implementation on it.
