# Existing release posts

## Goal

Make `Create post` / `relio post` generate announcement text from existing release tags, not only from unreleased commits. A fully committed and pushed project should still be able to create posts for prior releases.

## Tasks

- [x] Add release selection/defaulting for post generation.
- [x] Build post render plans from tagged release ranges.
- [x] Update tests and docs.
- [x] Run focused and full checks.
- [x] Show the interactive release picker with the same boxed version/date/commit table style as List releases.

## Evidence

- Created from user feedback that `Create post` showed nothing after all changes were committed and pushed.
- `relio post` now defaults to the latest tag outside a TTY, prompts for an existing release in the interactive menu/TTY path, and accepts `--version`.
- Verification passed: `go test ./cmd ./internal/i18n ./internal/menu` and `go test ./...`.
- Follow-up: release picker now uses a dedicated boxed TUI table matching `List releases`: version, date, commit count. Verification passed: `go test ./cmd` and `go test ./...`.
