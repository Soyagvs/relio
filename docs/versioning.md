# How the version is chosen

A version looks like `MAJOR.MINOR.PATCH` (for example `1.4.0`). The idea behind
[SemVer](https://semver.org/) is simple: the number tells the next person **how
risky the upgrade is**.

| Part change | Means | Example |
| ----------- | ----- | ------- |
| **MAJOR** | something you relied on changed or was removed — read the notes before upgrading | `1.3.2 → 2.0.0` |
| **MINOR** | new things were added, old things still work | `1.3.2 → 1.4.0` |
| **PATCH** | bug fixes only, safe to take | `1.3.2 → 1.3.3` |

Relio reads the commits since the last tag and picks the bump for you:

| Commits found | Bump | Example |
| ------------- | ---- | ------- |
| a `feat!:` or a `BREAKING CHANGE:` footer | **MAJOR** | `1.3.2 → 2.0.0` |
| any `feat:` (and no breaking change) | **MINOR** | `1.3.2 → 1.4.0` |
| any `fix:` / `perf:` / `refactor:` | **PATCH** | `1.3.2 → 1.3.3` |
| commits, but no conventional signal | **PATCH** | `1.3.2 → 1.3.3` |
| no commits since the last tag | — | *nothing to release* |

`--patch` / `--minor` / `--major` override the inference; the wizard's
"Change to…" options do the same interactively.

> [!TIP]
> Pre-1.0 (`0.x.y`): the API is not stable yet, so breaking changes inside a
> MINOR are conventionally acceptable — Relio still bumps MAJOR if you ask it to.

## Pre-releases

A **release candidate** lets you cut `v1.6.0-rc.1`, hand it to testers, iterate,
and only then ship `v1.6.0` — without inventing throwaway version numbers.

`relio --rc` targets a candidate instead of the final version. Because a bare
`relio` in a terminal always opens the menu, use `relio --rc --yes` to cut one
non-interactively, or pick **Release → Release candidate** in the menu. The
typical flow:

1. On stable `v1.5.0` with a `feat:` since, `relio --rc` cuts **`v1.6.0-rc.1`**
   — the version the commits imply, with `-rc.1` appended.
2. More commits land — run it again and the counter advances:
   **`v1.6.0-rc.2`** (same core version, `rc.1` → `rc.2`).
3. Ready to ship — `relio` with **no `--rc`** on an rc *finalizes* it:
   `v1.6.0-rc.2` → **`v1.6.0`**. The final `v1.6.0` changelog section summarises
   the **whole span since the last stable tag** — every commit across `rc.1`,
   `rc.2`, and anything after.
4. If a commit since the last rc escalates the target (a `feat!`, say),
   `relio --rc` moves the core up and restarts the counter: **`v2.0.0-rc.1`**.

`relio status` prints a hint line whenever `HEAD` is on a pre-release, telling
you the finalize version and the next-rc command.

The pre-release value carries all the way through: the git tag, the changelog
section heading, and any [version files](version-sync.md).

```bash
relio --rc --yes     # cut / advance the release candidate
relio --yes          # finalize the current rc to its stable core
```

When you [publish](publishing.md) an rc, GoReleaser already does the
right thing: `.goreleaser.yaml` carries `prerelease: auto` (marks a `-` tag as a
GitHub pre-release) and `skip_upload: auto` (skips the Homebrew tap bump) — no
config change needed.
