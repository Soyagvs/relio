# Dev build update notice

## Goal
Allow development builds (`version == "dev"`) to show the latest published Relio version when update checking is enabled, instead of suppressing the notice because there is no comparable current SemVer.

## Tasks
- [x] Adjust update availability logic for dev builds.
- [x] Cover dev-build update visibility and kill-switch behavior with tests.
- [x] Document dev-build/latest-version behavior.

## Evidence
- `go test ./internal/update` — pass.
- Dev builds now remain eligible for update checks unless `RELIO_NO_UPDATE_CHECK` or `CI` is set.
- For `current == "dev"`, a valid cached or freshly refreshed latest release is returned directly, without SemVer comparison.
- `RELIO_UPDATE_CHECK_ALWAYS=1` refreshes inline for dev builds and returns the freshly fetched latest release.
