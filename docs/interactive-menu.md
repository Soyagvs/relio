# The interactive menu

Running `relio` with no arguments in a terminal opens a one-shot menu — it runs
**one action and exits**, so the result stays on screen. Run `relio` again for
another action.

```
  RELIO menu

  1  Create release   Draft, preview, and tag a final release or rc
  2  Release status   See what's unreleased and the version it suggests
  3  Check commits    Validate Conventional Commits since the last tag
  4  List releases    Browse versions, read notes, delete one
  5  Create post      Create announcement text from a release
  6  Create image     Create a PNG card from a release
  7  Auth             GitHub connection — status and how to link
  8  Setup project    Create or inspect .release.yaml
  9  Settings         Language and release-footer preferences
 10  Guide            Step-by-step walkthrough of the whole flow
 11  Help             Every command and flag
 12  Exit             Leave Relio

  ↑/↓/j/k move · ? Help · g Guide · enter select · q quit
```

| Key | Does |
| --- | ---- |
| `↑` / `↓` / `j` / `k` | move the cursor |
| `?` | open the **Help** module |
| `g` | open the **Guide** module |
| `enter` | run the highlighted row |
| `q` | quit |

A few rows do more than run a flagless command:

- **Create release** asks three quick questions — *final release or release
  candidate*, *tag locally or push + publish*, and *whether to edit the notes
  first* — so you never have to remember `--rc`, `--publish`, or `--edit`. Then
  it runs the normal preview + wizard.
- **Auth** shows which GitHub token Relio found and who it belongs to, or
  explains how to connect one.
- **List releases**, **Create post**, and **Create image** use the same release
  table. The table shows versions only; press `Enter` to open a release preview,
  then `b` / `Esc` to return to the table. List releases can preview and delete
  local releases with `d`. Create post/image move to the `✔ confirm` button and
  press `Enter`, then continue to their format or image options.
- **Create image** can save the PNG, upload it, or both. Upload paths print a
  plain URL first, then a QR code, so mobile terminals and `mosh` sessions have
  a copyable link even when QR scanning is awkward.
- **Help** opens a tabbed reference instead of printing one long page. Use
  `←`/`→` or `tab` to switch between command, flag, and menu sections, then
  `Enter` on `Back to home` to return to the menu.
- **Setup** runs `relio init` (or tells you the config already exists).
- **Settings** switches the UI language (English / Español) and toggles the
  changelog-footer preferences (contributors line, compare link) — see
  [Language](language.md) for details. It's menu-only, there's no equivalent flag.

The design intent, in one line: **anything the flags can do, the menu can do.**
The flags are the scripting surface; the menu is the interactive one.

The menu is skipped — and a release runs directly — when the output is not a
terminal (CI, a pipe), or when you pass `-y` / `--yes`, or a forced bump
(`--patch` / `--minor` / `--major`). `--rc` and `--publish` on their own still
open the menu; combine them with `--yes` to act without prompts.

The banner shows the current version, and — when a newer Relio has been
published — `▲ vX.Y.Z available` right below it. The check reads one cached value
on disk and, at most once a day, refreshes it in the background; it never blocks
the menu or sends anything about you. `relio version` shows the same hint. Set
`RELIO_NO_UPDATE_CHECK=1` (or run in CI) to turn it off.
