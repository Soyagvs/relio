# How the changelog is built

Each commit's Conventional Commit **type** maps to a Keep-a-Changelog section:

| Commit type | Changelog section |
| ----------- | ----------------- |
| `feat` | **Added** |
| `fix` | **Fixed** |
| `perf`, `refactor`, `revert`, `style` | **Changed** |
| `docs`, `chore`, `test`, `build`, `ci`, unknown | *left out of the changelog* |

A `BREAKING CHANGE:` footer or a `!` before the colon also adds the line to
**Changed**, prefixed with `**Breaking:**`. The scope, if any, is kept:
`fix(kiosk): header alignment` → `- kiosk: Header alignment`.

Run `relio --edit` to hand-edit the result before it is written. Relio opens
`$RELIO_EDITOR` / `$VISUAL` / `$EDITOR` (falling back to `vi`) on the generated
notes body — everything under the `## [x.y.z] - date` heading. Save your changes
to use them, or save an empty or unchanged buffer to keep the generated notes.
The edited text is written verbatim to `CHANGELOG.md` (under a freshly rendered
heading) and used as the GitHub Release body. The heading itself is not editable.
The optional footer described below is added after your edited notes, not shown
in the editor.

## Contributors and a compare link

Two opt-in toggles under `release:` add a trailing block to every changelog
section. `contributors: true` appends a line such as `Thanks to Alice, Bob.`,
built from the `git log` author names of the commits in the release range, in
first-seen order, with any name ending in `[bot]` filtered out — these are git
author names, not GitHub `@handles`. `compare_link: true` appends
`**Full changelog**: https://github.com/owner/repo/compare/v1.5.0...v1.6.0`.

Both lines also land in the GitHub Release body. The compare link needs a
GitHub `origin` remote (or an explicit `github.repo`) to resolve `owner/repo`,
and a previous tag to compare against, so it is skipped on the first release.
When both toggles are off, or neither line can be built, no footer is added.

While both are off, a `relio` release ends with a dimmed one-line reminder that
these toggles exist, so the feature stays discoverable without reading the docs.
The reminder disappears once either toggle is set.
