# Update check always mode

## Goal
Add an opt-in mode so Relio can query the latest release on every interactive entry instead of waiting for the 24h cache TTL.

## Tasks
- [x] Add an environment switch for always-refresh update checks.
- [x] Cover fresh-cache and disabled/dev behavior with tests.
- [x] Document the opt-in behavior in user docs.

## Evidence
- `RELIO_UPDATE_CHECK_ALWAYS` makes `internal/update.Available` refresh inline and return the refreshed latest value instead of using a fresh lower cache.
- `RELIO_NO_UPDATE_CHECK`, `CI`, and unparseable/dev current versions still short-circuit before any refresh.
- Focused validation: `go test ./internal/update` passed.
- Broader validation: `go test ./...` passed.
