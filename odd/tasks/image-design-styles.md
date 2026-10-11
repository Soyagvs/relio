# Image design styles

## Goal
Rework `relio image` so the interactive flow asks for format first, then design style, then that design's colour theme. Keep today's renderer under the `default` design and add `minimal`, `bold`, and `neon` designs with their own colour themes.

## Tasks
- [x] Add card design/style model and render dispatch.
- [x] Add CLI/picker flow for design → theme while preserving existing flags.
- [x] Add localized labels, docs, and tests for designs/themes.
- [x] Verify image rendering and command tests.

## Evidence
- `go test ./internal/card` passed.
- `go test ./cmd -run 'Image|Help'` passed.
- `go test ./internal/i18n` passed.
- `go test ./...` passed.
- Default design keeps the legacy `relio-<version>-<shape>.png` filename; non-default designs include design and theme to avoid overwriting variants.
