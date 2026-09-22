# Release hooks

Run your own shell commands around a release by listing them under
`release.hooks` — a lint pass before the preview, a smoke test before writing,
a notification after, whatever the project needs.

```yaml
release:
    hooks:
        validate: make lint                     # runs before the plan preview is even shown
        before: make test                       # one command…
        after:                                   # …or a list, run in order
            - ./scripts/changelog-to-slack.sh
            - echo "shipped $RELIO_TAG"
```

Each hook is a single shell command string, or a list of them. A list stops at
the first command that exits non-zero.

| Hook | Runs | A non-zero exit… |
| ---- | ---- | ---------------- |
| **`validate`** | before the plan preview is shown / before you're asked to confirm | **aborts** the release — no changelog, no commit, no tag — and `relio` exits non-zero |
| **`before`** | after you confirm the release, before *anything* is written | **aborts** the release — no changelog, no commit, no tag — and `relio` exits non-zero |
| **`after`** | at the very end: after the tag, and after the `--publish` push + GitHub Release when that ran | prints a **warning** only; `relio` still exits 0, because the release already happened |

Hooks run through the platform shell (`sh -c` on Unix, `cmd /c` on Windows) with
the **repository root** as the working directory. Output is streamed straight
through. They inherit the process environment plus three extra variables:

| Variable | Example | Meaning |
| -------- | ------- | ------- |
| `RELIO_VERSION` | `1.6.0` | the new version, no `v` |
| `RELIO_TAG` | `v1.6.0` | the new tag, honouring `tag_prefix` |
| `RELIO_PREVIOUS_TAG` | `v1.5.0` | the tag the release was computed from (empty on a first release) |

Pass `--no-hooks` to skip every hook for one run. `--yes` does **not** skip
hooks — CI needs them to run. Hooks never run for `relio status` or `relio check`
(they apply nothing).
