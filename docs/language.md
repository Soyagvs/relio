# Language

Relio's interactive screens, `--help`, and command output can run in English or
Español. Pick a language from the menu's **Settings** entry (`relio` → `9`) — it
applies immediately and is saved for next time. There is no flag; the language
is resolved once, before any command runs:

1. `.release.yaml`'s `language:` key at the repository root, if set (per-repo
   override — currently edit this file by hand, there's no UI writer for it yet)
2. the global preference saved by the Settings screen
3. `en`, if neither is set

The global preference lives outside any repository, in a small `config.yaml`
Relio manages for you:

| OS | Path |
| --- | ---- |
| Linux | `$XDG_CONFIG_HOME/relio/config.yaml` (falls back to `~/.config/relio/config.yaml`) |
| macOS | `~/Library/Application Support/relio/config.yaml` |
| Windows | `%AppData%\relio\config.yaml` |

A missing or unreadable file is never an error — Relio just runs in English.
Content Relio writes to disk — `CHANGELOG.md` sections, GitHub Release bodies,
and `relio post` output — always stays in English regardless of the UI
language, so generated artifacts read consistently for every audience.

Translation coverage grows over time; any string not yet translated for a
language falls back to its English text rather than showing a raw key.
