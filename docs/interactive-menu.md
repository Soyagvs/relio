# The interactive menu

Running `relio` with no arguments in a terminal opens a one-shot menu — it runs
**one action and exits**, so the result stays on screen. Run `relio` again for
another action.

```
  RELIO menu

  1  Release          Create a release — final or rc, and optionally push + publish
  2  Status           What's unreleased and the version it suggests
  3  Check            Which commits since the last tag are Conventional Commits
  4  Releases         Browse versions, read notes, delete one
  5  Announcement     Copy-paste release text — pick a format
  6  Release image    Save or share a PNG release card
  7  Auth             GitHub connection — status and how to link
  8  Setup            Create or inspect .release.yaml
  9  Settings         Language and release-footer preferences
     Guide            Step-by-step walkthrough of the whole flow
     Help             Every command and flag
     Exit             Leave Relio

  ↑/↓ move · 1–9 jump · ? help · g guide · enter select · q quit
```

| Key | Does |
| --- | ---- |
| `↑` / `↓` | move the cursor |
| `1`–`9` | jump straight to that row and select it (the first nine rows only — **Guide**, **Help**, and **Exit** are arrow-only) |
| `?` | open **Help** |
| `g` | open the **Guide** |
| `enter` | run the highlighted row |
| `q` | quit |

A few rows do more than run a flagless command:

- **Release** asks three quick questions — *final release or release candidate*,
  *tag locally or push + publish*, and *whether to edit the notes first* — so you
  never have to remember `--rc`, `--publish`, or `--edit`. Then it runs the
  normal preview + wizard.
- **Auth** shows which GitHub token Relio found and who it belongs to, or
  explains how to connect one.
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
