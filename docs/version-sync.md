# Syncing the version into project files

Plenty of projects keep the version in more than one place — `package.json`,
`pyproject.toml`, a `VERSION` file. List those files under
`release.version_files` and Relio rewrites each one with the new version **in the
same `chore(release): vX.Y.Z` commit** as the changelog, so the tag points at a
commit where everything agrees.

```yaml
release:
    version_files:
        - package.json                       # known filename — built-in rule
        - VERSION                             # whole-file version token
        - Cargo.toml                          # first `version = "…"` line
        - { path: src/app/__init__.py, pattern: '__version__ = "([^"]+)"' }
```

**Known filenames need no pattern:**

| File | Rule |
| ---- | ---- |
| `package.json` | the `"version": "…"` value |
| `Cargo.toml`, `pyproject.toml`, any `*.toml` | the first `^version = "…"` line (single or double quotes) |
| `setup.py` | the `version="…"` keyword argument |
| `Chart.yaml`, `Chart.yml` | the `^version:` line (never Helm's separate `appVersion:`) |
| `VERSION`, `version.txt` | the file's lone version token |

**Anything else** takes a `{ path, pattern }` entry whose `pattern` is a Go
regexp with **exactly one capture group** wrapping the version substring.

The value written is the **bare number**, no leading `v` (`1.6.0`). Replacement
is regex-based — Relio swaps the matched span and leaves every other byte
(indentation, key order, trailing newline) untouched; it never reformats the
file. A file already at the target version is a no-op.

A **missing file** or a **pattern that does not match** fails the run *before
anything is written*. Pass `--no-version-files` to skip the whole step for one
run.
