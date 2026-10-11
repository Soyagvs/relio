# Shared release selector

## Goal

Unify the release-selection UX used by List releases, Create post, and Create image.

## Desired behavior

- All three flows use the same release table component.
- The table has no inline preview/detail pane.
- Pressing Enter on a release opens a release preview/detail screen.
- From preview, the user can go back to the table.
- In List releases: preview/detail allows viewing and deleting only.
- In Create post: preview/detail allows confirming the selected release, then asks for post format and prints the post.
- In Create image: preview/detail allows confirming the selected release, then continues with shape/theme/destination flow.

## Tasks

- [x] Extract or introduce a shared release table/preview selector.
- [x] Wire List releases to use it with view/delete actions.
- [x] Wire Create post to use it with preview/confirm before format selection.
- [x] Wire Create image to use it with preview/confirm before image options.
- [x] Update tests and docs.
- [x] Run focused and full checks.

## Evidence

- Created from user request to make Create post, Create image, and List releases share the same table and preview/confirm flow.
- Implemented shared table/preview flow in `internal/releases/releases.go` and reused it from `cmd/post.go` and `cmd/image.go`.
- Verification passed: `go test ./internal/releases ./cmd` and `go test ./...`.
