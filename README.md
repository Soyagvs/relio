# Relio

Turn a repository's git activity into a **version, changelog, and tag** — in one
command, with a preview before anything is written.

```
code  →  commit  →  push  →  relio  →  version + changelog + tag
```

Relio does not try to replace git or GitHub. It removes the repetitive work
that happens *after* you finish coding.

_created by SOYAGVS_

---

## Install

```bash
go install github.com/soyagvs/relio@latest   # installs the `relio` binary

# from source
git clone https://github.com/soyagvs/relio
cd relio
go build -o relio .
```

## Quick start

```bash
relio init     # writes .release.yaml (config only, never secrets)
relio          # opens the interactive menu
```

## Main menu

Running `relio` with no arguments in a terminal opens the menu:

```
  ██████╗ ███████╗██╗     ██╗ ██████╗       (REL orange, IO purple)
  ██╔══██╗██╔════╝██║     ██║██╔═══██╗
  ██████╔╝█████╗  ██║     ██║██║   ██║
  ██╔══██╗██╔══╝  ██║     ██║██║   ██║
  ██║  ██║███████╗███████╗██║╚██████╔╝
  ╚═╝  ╚═╝╚══════╝╚══════╝╚═╝ ╚═════╝
        turn commits into releases        (purple)
  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
        created by SOYAGVS

▸ Create a release
    Version, changelog, and tag from commits since the last tag
  Releases
    List every version, read its notes, or delete one
  Release text
    Copy-paste announcement for social posts — pick a format
  Release image
    Save a shareable PNG of a release — pick a shape
  GitHub auth
    Log in with your own GitHub account (coming in v0.2.0)
  Help
    Every command and flag, with a one-line description
  Exit
    Leave Relio

↑/↓ move · enter select · q quit
```

The menu runs **one action and exits** — the result stays on screen. Run
`relio` again for another action.

**Releases** opens a browser of every tag: move with `↑/↓` to read each
version's notes (with commit hashes), press `d` to delete one (removes the git
tag **and** its `CHANGELOG.md` section, after a `y/N` confirm), `q` to go back.

**Release text** asks which format you want, then prints the announcement.

**Release image** asks for a release and a shape (horizontal / vertical / square) and
writes a PNG card into the current directory.

In CI or when piped (no TTY), or with `--yes` / a forced bump, the menu is
skipped and a release runs directly.

## What "Create a release" looks like

```
⬢ Relio  v1.4.0  ·  azeink

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
relio --patch
relio --minor
relio --major
```

## Commands

| Command            | What it does                                                   |
| ------------------ | ------------------------------------------------------------- |
| `relio`       | Open the menu, or run a release directly in CI / with `--yes` |
| `relio init`  | Scaffold `.release.yaml`                                      |
| `relio post`  | Generate an announcement (`--format minimal\|social\|technical\|casual\|changelog`) |
| `relio image` | Save a shareable PNG of a release (`--shape horizontal\|vertical\|square`) |
| `relio auth`  | GitHub login — **placeholder, lands in v0.2.0**              |
| `relio version` | Print the tool version                                     |

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
  pick/              reusable Bubble Tea single-select
  card/              PNG release image (fogleman/gg + Go fonts)
```

## Roadmap

- **v0.1.0** — local git tool: parse, version, changelog, tag, preview *(current)*
- **v0.2.0** — GitHub: OAuth Device Flow, keychain storage, push tag, create Release
- **v0.3.0** — content: `relio post` templates, clipboard
- **v0.4.0** — plugin API (`BeforeRelease` / `AfterRelease` / `OnTagCreated` / `OnReleasePublished`)
- **v1.0.0** — `relio init` → `relio auth login` → `relio`, polished

## Development

```bash
go test ./...
go build -o relio .
```
