# Go Release

Turn a repository's git activity into a **version, changelog, and tag** — in one
command, with a preview before anything is written.

```
code  →  commit  →  push  →  go-release  →  version + changelog + tag
```

Go Release does not try to replace git or GitHub. It removes the repetitive work
that happens *after* you finish coding.

_created by SOYAGVS_

---

## Install

```bash
go install github.com/soyagvs/go-release@latest   # installs the `go-release` binary

# from source
git clone https://github.com/soyagvs/go-release
cd go-release
go build -o go-release .
```

## Quick start

```bash
go-release init     # writes .release.yaml (config only, never secrets)
go-release          # opens the interactive menu
```

## Main menu

Running `go-release` with no arguments in a terminal opens the menu:

```
╔═══════════════════════════════════════════════════════════════════════════╗
║   ██████╗  ██████╗    ██████╗ ███████╗██╗     ███████╗ █████╗ ███████╗    ║
║   ██╔════╝ ██╔═══██╗   ██╔══██╗██╔════╝██║     ██╔════╝██╔══██╗██╔════╝    ║
║   ██║  ███╗██║   ██║   ██████╔╝█████╗  ██║     █████╗  ███████║███████╗    ║
║   ╚██████╔╝╚██████╔╝   ██║  ██║███████╗███████╗███████╗██║  ██║███████║    ║
║    ╚═════╝  ╚═════╝    ╚═╝  ╚═╝╚══════╝╚══════╝╚══════╝╚═╝  ╚═╝╚══════╝    ║
║                  from finished code to a published release                ║
╚═══════════════════════════════════════════════════════════════════════════╝
  created by SOYAGVS

▸ Create a release
    Version, changelog, and tag from commits since the last tag
  Releases
    List every version, read its notes, or delete one
  Release text
    Copy-paste announcement for social posts — pick a format
  GitHub auth
    Log in with your own GitHub account (coming in v0.2.0)
  Help
    Every command and flag, with a one-line description
  Exit
    Leave Go Release

↑/↓ move · enter select · q quit
```

The menu runs **one action and exits** — the result stays on screen. Run
`go-release` again for another action.

**Releases** opens a browser of every tag: move with `↑/↓` to read each
version's notes (with commit hashes), press `d` to delete one (removes the git
tag **and** its `CHANGELOG.md` section, after a `y/N` confirm), `q` to go back.

**Release text** asks which format you want, then prints the announcement.

In CI or when piped (no TTY), or with `--yes` / a forced bump, the menu is
skipped and a release runs directly.

## What "Create a release" looks like

```
⬢ Go Release  v1.4.0  ·  azeink

╭──────────────────────────╮
│  Current version v1.3.2  │
│  Detected change minor   │
│  Next version    v1.4.0  │
╰──────────────────────────╯

Added
  • Add transaction categories
Fixed
  • Dashboard crash on empty state

▸ Create v1.4.0
  Change to patch
  Change to minor
  Change to major
  Cancel

↑/↓ move · enter select · y confirm · q cancel
```

On confirm:

```
✓ CHANGELOG.md updated
✓ git tag v1.4.0 created
✓ release ready

  next:  git push && git push origin v1.4.0
```

## How the version is chosen

With [Conventional Commits](https://www.conventionalcommits.org/):

| Commit                     | Bump  | Example         |
| -------------------------- | ----- | --------------- |
| `fix:` / `perf:` / `refactor:` | PATCH | `1.3.2 → 1.3.3` |
| `feat:`                    | MINOR | `1.3.2 → 1.4.0` |
| `feat!:` or `BREAKING CHANGE:` | MAJOR | `1.3.2 → 2.0.0` |

Override it:

```bash
release --patch
release --minor
release --major
```

## Commands

| Command            | What it does                                                   |
| ------------------ | ------------------------------------------------------------- |
| `go-release`       | Open the menu, or run a release directly in CI / with `--yes` |
| `go-release init`  | Scaffold `.release.yaml`                                      |
| `go-release post`  | Generate an announcement (`--format technical\|casual\|changelog`) — experimental |
| `go-release auth`  | GitHub login — **placeholder, lands in v0.2.0**              |
| `go-release version` | Print the tool version                                     |

### Useful flags

| Flag             | Meaning                                        |
| ---------------- | -------------------------------------------- |
| `-y, --yes`      | Skip the wizard (required in CI / non-TTY)   |
| `-C, --dir path` | Run as if started in `path`                  |
| `--no-changelog` | Do not touch the changelog file             |
| `--no-tag`       | Do not create the git tag                    |

## Configuration — `.release.yaml`

```yaml
project: azeink
versioning: semver        # only "semver" for now
commits: conventional     # only "conventional" for now
release:
    changelog: true
    changelog_file: CHANGELOG.md
    tag: true
    tag_prefix: v
github:
    enabled: false        # v0.2.0
content:
    enabled: false        # v0.3.0
```

This file holds **configuration only**. Credentials never go here — the future
GitHub integration will store per-user tokens in the OS keychain.

## Project layout

```
main.go
cmd/                 Cobra command wiring
internal/
  conventional/      Conventional Commits parser
  semver/            version parsing + bump rules
  changelog/         Keep a Changelog rendering
  config/            .release.yaml load/save
  gitrepo/           thin wrapper over the git binary
  release/           orchestration: build a plan, apply it
  ui/                lipgloss palette + banner + non-interactive views
  menu/              Bubble Tea main menu
  wizard/            Bubble Tea confirmation step
  releases/          Bubble Tea release browser (view / delete versions)
```

## Roadmap

- **v0.1.0** — local git tool: parse, version, changelog, tag, preview *(current)*
- **v0.2.0** — GitHub: OAuth Device Flow, keychain storage, push tag, create Release
- **v0.3.0** — content: `release post` templates, clipboard
- **v0.4.0** — plugin API (`BeforeRelease` / `AfterRelease` / `OnTagCreated` / `OnReleasePublished`)
- **v1.0.0** — `init` → `auth login` → `release`, polished

## Development

```bash
go test ./...
go build -o release .
```
