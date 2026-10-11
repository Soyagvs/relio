# Release browser table pagination

## Goal

Keep orientation in `relio releases` when there are many releases by showing a lazigit-style split view: a compact release table on the left and the selected release summary on the right. The left table shows version, date, and commit count; Enter moves focus into the summary pane so long notes can be scrolled; Esc/Tab returns to the release table; q/Esc from the table returns to the menu.

## Tasks

- [x] Map current release browser data and tests.
- [x] Implement release metadata table and pagination controls.
- [x] Add or update tests for table content and paging behavior.
- [x] Run focused and full Go checks.
- [x] Redesign release browser as a two-pane table/detail view.
- [x] Add focused tests for pane focus and summary scrolling.
- [x] Run focused and full Go checks for the split-pane UI.
- [x] Fix q to return cleanly to menu without leaving a static release list.
- [x] Change pane navigation to lazigit-style h/l focus movement.
- [x] Fix menu return so `q` in releases redraws the main menu immediately.
- [x] Fix split-pane alignment with ANSI-aware padding.
- [x] Update docs for release browser navigation and image link output.

## Evidence

- Created from user request in current session.
- Implemented by `gentle-ai-worker`; parent adjusted release-browser copy to use existing i18n and `AuthorsBetween`.
- Verification passed: `go test ./internal/releases ./internal/gitrepo ./internal/i18n` and `go test ./...`.
- Follow-up requested from screenshot: current single-column layout is messy; user wants side-by-side release table and selected summary.
- Split-pane follow-up verification passed: `go test ./internal/releases ./internal/i18n` and `go test ./...`.
- Final navigation polish: `q` leaves no static list in scrollback; `l` enters details, `h`/Esc/Tab returns to versions, `n/p` pages the release table. Verification passed: `go test ./internal/releases ./internal/i18n` and `go test ./...`.
- Follow-up fix: removed the post-releases wait/back picker so `q` returns straight to the menu, and changed split-pane joining to use `lipgloss.Width`-aware padding so styled table rows do not break alignment. Verification passed: `go test ./internal/releases ./cmd ./internal/i18n` and `go test ./...`.
- Documentation updated in `docs/commands.md` and `docs/interactive-menu.md` for the split releases browser and plain image URL output.
- Work-unit commit: `602c388` (`fix: polish release browser and image links`) on branch `fix/releases-image-ui`, pushed to `origin/fix/releases-image-ui`.
