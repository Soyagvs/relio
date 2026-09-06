<!-- Hero image goes here. Drop a file at docs/hero.png (≈720px wide). -->
<p align="center">
  <img src="docs/hero.png" alt="Relio" width="720">
</p>

<h1 align="center">Relio</h1>

<p align="center"><i>turn commits into releases</i></p>

Relio reads a repository's git activity and turns it into a **version, a
changelog, and a tag** — with a preview before anything is written. It does not
try to replace git or GitHub; it removes the repetitive work that happens
*after* you finish coding.

```
code  →  commit  →  push  →  relio  →  version + CHANGELOG.md + tag
```

Everything Relio produces comes from the real release: the commits since your
last tag, parsed as [Conventional Commits](https://www.conventionalcommits.org/).

---

## Contents

- [Install](#install)
- [Quick start](#quick-start)
- [The interactive menu](#the-interactive-menu)
- [Commands](#commands)
  - [`relio`](#relio--create-a-release) · [`relio status`](#relio-status) · [`relio init`](#relio-init) · [`relio post`](#relio-post) · [`relio image`](#relio-image) · [`relio auth`](#relio-auth) · [`relio version`](#relio-version)
- [Global flags](#global-flags)
- [How the version is chosen](#how-the-version-is-chosen)
- [How the changelog is built](#how-the-changelog-is-built)
- [Configuration — `.release.yaml`](#configuration--releaseyaml)
- [Non-interactive / CI usage](#non-interactive--ci-usage)
- [Project layout](#project-layout)
- [Roadmap](#roadmap)
- [Contributing](#contributing)
- [License](#license)

---

## Install

```bash
# with Go (installs the `relio` binary onto $GOPATH/bin or $GOBIN)
go install github.com/soyagvs/relio@latest

# from source
git clone https://github.com/soyagvs/relio
cd relio
go build -o relio .          # ./relio
# or: make install           # builds to ~/.cargo/bin/relio
```

Requires **Go 1.22+** and the `git` binary on `PATH`.

---

## Quick start

```bash
cd your-project
relio init        # writes .release.yaml (configuration only, never secrets)
relio             # opens the interactive menu
```

Pick **Create a release**, check the preview, confirm. Relio updates
`CHANGELOG.md`, creates an annotated git tag, and tells you the `git push` to
run. **Nothing is pushed for you.**

---

## The interactive menu

Running `relio` with no arguments in a terminal opens a one-shot menu — it runs
**one action and exits**, so the result stays on screen. Run `relio` again for
another action.

```
  ██████╗ ███████╗██╗     ██╗ ███████╗
  ██╔══██╗██╔════╝██║     ██║████ ████
  ██████╔╝█████╗  ██║     ██║███ ▪ ███
  ██╔══██╗██╔══╝  ██║     ██║███   ███
  ██║  ██║███████╗███████╗██║████ ████
  ╚═╝  ╚═╝╚══════╝╚══════╝╚═╝ ███████
        turn commits into releases
  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

▸ Status              What's unreleased since the last tag, and the suggested version
  Create a release    Version, changelog, and tag from commits since the last tag
  Releases            List every version, read its notes, or delete one
  Release text        Copy-paste announcement for social posts — pick a format
  Release image       Save a shareable PNG of a release — pick a shape
  GitHub auth         Log in with your own GitHub account (coming in v0.2.0)
  Help                Every command and flag, with a one-line description
  Exit

↑/↓ move · enter select · q quit
```

In CI or when the output is piped (no TTY), or with `--yes` / a forced bump, the
menu is skipped and a release runs directly.

---

## Commands

### `relio` — create a release

With no subcommand, `relio` builds a **release plan** from the commits since the
last tag and applies it.

1. Find the latest tag (`git describe --tags --abbrev=0`); if there is none,
   start from `v0.0.0`.
2. Read `<tag>..HEAD`, parse each commit as a Conventional Commit.
3. Decide the next version (see [below](#how-the-version-is-chosen)).
4. Show the preview — **nothing is written yet**:

   ```
   ⬢ Relio  v0.4.0  ·  azeink

   ╭──────────────────────────╮
   │  Current version v1.3.2  │
   │  Detected change minor   │
   │  Next version    v1.4.0  │
   ╰──────────────────────────╯

   Added
     a967103  Add transaction categories
   Fixed
     92af81e  Dashboard crash on empty state

   3 commits since v1.3.2
   ```

5. In a terminal you get a small wizard: **Create**, change the bump
   (patch / minor / major), or cancel. With `--yes` it applies immediately.
6. On confirm:

   ```
   ✓ CHANGELOG.md updated
   ✓ git tag v1.4.0 created
   ✓ release ready

     next:  git push && git push origin v1.4.0
   ```

`CHANGELOG.md` gets a new `## [x.y.z] - YYYY-MM-DD` section at the top of the
release history, in the [Keep a Changelog](https://keepachangelog.com/) format.
The git tag is **annotated** and created at `HEAD`.

| Flag                        | Meaning |
| --------------------------- | ------- |
| `--patch` `--minor` `--major` | Force the bump instead of inferring it from the commits (only one at a time). |
| `-y, --yes`                  | Skip the menu and the confirmation. Required in CI / a non-interactive shell. |
| `--no-changelog`             | Do not touch the changelog file. |
| `--no-tag`                   | Do not create the git tag. |

```bash
relio                 # interactive
relio --yes           # apply the inferred bump, no prompts
relio --minor --yes   # force a minor bump
relio --no-tag        # write the changelog only
```

---

### `relio status`

A read-only glance at what's unreleased and the version it suggests. No prompts,
safe to run anywhere.

```
$ relio status

azeink

Current      v1.4.0
Unreleased   6 commits

feat      3
fix       2
refactor  1

Suggested    v1.5.0  (minor)

! uncommitted changes in the working tree
Ready to release.
```

When there are no new commits: `Suggested —` and `Nothing to release.`
Also available as the **Status** menu entry.

---

### `relio init`

Writes `.release.yaml` at the repository root with sensible defaults. The file
holds **configuration only** — credentials never go in it. Refuses to overwrite
an existing file.

| Flag        | Meaning |
| ----------- | ------- |
| `--project` | Project name. Defaults to the `origin` remote's repo name, else the directory name. |

```bash
relio init
relio init --project azeink
```

---

### `relio post`

Generates a short, plain-text announcement from the commits since the last tag.
**Only the text goes to stdout**, so `relio post | pbcopy` (or `| wl-copy`)
copies it cleanly. Colour is added when stdout is a terminal and stripped when
it is piped.

| `--format`   | Output |
| ------------ | ------ |
| `minimal` *(default)* | Byte-identical to what the **Releases** browser prints for a version: `relio -- release`, `<project> · <version>`, `<date> · <time> · N commits`, then the grouped notes. |
| `social`     | Shortest. `Project -- Release`, then `version · DD.MM.YY · HH:MM`, then one `<type>  <description>` line per **notable** commit (`feat`, `fix`, `perf`, `refactor`, `revert`, `style`). |
| `technical`  | Terse `•` bullet list — good for a changelog or a dev channel. |
| `casual`     | Loose tone: `proj v1.4.0 is out. → …` |
| `changelog`  | The exact section that goes into `CHANGELOG.md`. |

```
$ relio post --format social
Azeink -- Release

v1.4.0 · 06.09.26 · 14:36

feat      Add facial attendance
feat      Add new kiosk interface
refactor  Authentication flow
fix       Supervisor login
```

*(This is a preview of the v0.3.0 content generator — nothing is published.)*

---

### `relio image`

Renders a dark, developer-styled **release card** as a PNG into the current
directory (`relio-<version>-<shape>.png`). Everything on it comes from the real
release. In a terminal it prompts for the release, the shape, the colour, and
whether to upload; the flags below skip those prompts (and prompts are skipped
automatically with no TTY — defaults are latest tag / horizontal / orange).

| Flag        | Values |
| ----------- | ------ |
| `--version` | Release tag to render. Default: the latest tag. |
| `--shape`   | `horizontal` (1200×630, Twitter/OG) · `vertical` (1080×1920, Instagram story) · `square` (1080×1080). Each has its **own** responsive layout, not a crop. |
| `--theme`   | Accent colour: `orange` *(default)* · `green` · `purple`. The section colours (Added green / Changed amber / Fixed coral) are fixed. |
| `--hash`    | Prefix each line with its short commit hash. Off by default. |
| `--upload`  | Also upload the PNG to a temporary public host (litterbox, 72h; catbox as fallback) and print the URL **plus a QR code**. Handy for getting it onto a phone over `mosh` — only text crosses the wire. The local file is always kept. |

```bash
relio image                                   # prompts for everything
relio image --shape square --theme green
relio image --version v1.4.0 --shape vertical --upload
```

If a changelog line is long it wraps onto the next line; if there are too many
commits for the card, each section is capped and a `+ N more` line is added — it
never spills outside the card, on any shape.

The **Releases** menu entry opens a browser of every tag:

```
↑/↓  move between versions        (the pane shows that version's notes)
enter  print the selected version's notes to the terminal and exit
d      delete the version — removes the git tag AND its CHANGELOG.md section (y/N)
q      back
```

---

### `relio auth`

Placeholder for the v0.2.0 GitHub integration (per-user OAuth Device Flow, tokens
in the OS keychain — never in `.release.yaml`). The subcommands
(`login` / `status` / `logout`) currently just print a "lands in v0.2.0" note.

---

### `relio version`

```
$ relio version
Relio v1.4.0 (commit a1b2c3d, built 2026-09-06)
```

The version is `dev` unless the binary was built with `HEAD` exactly on a tag
(that's what `make install` does), or with
`-ldflags "-X github.com/soyagvs/relio/cmd.version=…"`.

---

## Global flags

Available on every command:

| Flag              | Meaning |
| ----------------- | ------- |
| `-C, --dir <path>` | Run as if Relio was started in `<path>`. |
| `--no-hash`         | Hide the commit hash on each release-note line (preview, Releases browser, `post`). |

---

## How the version is chosen

Relio reads the commits since the last tag and applies
[SemVer](https://semver.org/): a version is `MAJOR.MINOR.PATCH`, and the number
tells someone how safe it is to upgrade.

| Commits found | Bump | Example |
| ------------- | ---- | ------- |
| a `feat!:` or a `BREAKING CHANGE:` footer | **MAJOR** | `1.3.2 → 2.0.0` |
| any `feat:` (and no breaking change) | **MINOR** | `1.3.2 → 1.4.0` |
| only `fix:` / `perf:` / `refactor:` | **PATCH** | `1.3.2 → 1.3.3` |
| commits, but no conventional signal | **PATCH** | `1.3.2 → 1.3.3` |
| no commits since the last tag | — | *nothing to release* |

`--patch` / `--minor` / `--major` override the inference; the wizard's
"Change to…" options do the same interactively.

> Pre-1.0 (`0.x.y`): the API is not stable yet, so breaking changes in a MINOR
> are conventionally acceptable — Relio still bumps MAJOR if you ask it to.

---

## How the changelog is built

Each commit's Conventional Commit **type** maps to a Keep-a-Changelog section:

| Commit type | Section |
| ----------- | ------- |
| `feat` | **Added** |
| `fix` | **Fixed** |
| `perf`, `refactor`, `revert`, `style` | **Changed** |
| `docs`, `chore`, `test`, `build`, `ci`, unknown | *omitted from the changelog* |

A `BREAKING CHANGE:` footer or a `!` before the colon also adds the line to
**Changed**, prefixed with `**Breaking:**`. The scope, if any, is kept:
`fix(kiosk): header alignment` → `- kiosk: Header alignment`.

---

## Configuration — `.release.yaml`

Written by `relio init`, read from the repository root. **Configuration only —
no secrets.**

```yaml
project: azeink            # shown on cards / posts; free text
versioning: semver         # only "semver" is supported
commits: conventional      # only "conventional" is supported

release:
    changelog: true              # update the changelog file on release
    changelog_file: CHANGELOG.md  # which file
    tag: true                    # create the git tag on release
    tag_prefix: v                # "" for bare 1.4.0 tags

github:
    enabled: false         # reserved for v0.2.0

content:
    enabled: false         # reserved for v0.3.0
```

Missing fields fall back to these defaults, so a minimal file with just
`project:` works. Unknown `versioning` / `commits` values are rejected with a
clear error.

---

## Non-interactive / CI usage

Relio detects a non-TTY and behaves predictably:

```bash
relio --yes                          # infer + apply, no prompts (fails without --yes)
relio --minor --yes --no-changelog   # force minor, tag only
relio status                         # read-only, exit 0
relio post --format changelog        # plain text on stdout
relio image --shape horizontal       # latest tag, default theme, saved to cwd
```

A minimal GitHub Actions job that cuts a release from the default branch:

```yaml
- uses: actions/checkout@v4
  with: { fetch-depth: 0 }        # tags + full history
- run: go install github.com/soyagvs/relio@latest
- run: relio --yes
- run: git push --follow-tags
```

Relio never pushes; wire the `git push` into your pipeline.

---

## Project layout

```
main.go
cmd/                 Cobra command wiring (root, status, init, post, image, auth, version)
internal/
  conventional/      Conventional Commits parser
  semver/            version parsing + bump rules
  changelog/         Keep a Changelog rendering, section extract/remove
  config/            .release.yaml load / save
  gitrepo/           thin wrapper over the git binary
  release/           orchestration: build a plan, apply it
  ui/                lipgloss palette, banner, non-interactive views
  menu/              Bubble Tea main menu
  wizard/            Bubble Tea release confirmation
  releases/          Bubble Tea release browser (view / delete)
  pick/              reusable Bubble Tea single-select
  card/              PNG release-card renderer (fogleman/gg + Go fonts)
  upload/            temporary file host client (litterbox / catbox)
```

`gitrepo` shells out to the system `git` — no CGO, no `go-git`.

---

## Roadmap

- **v0.1.x** — local git tool: parse, version, changelog, tag, preview · status · post · image *(current)*
- **v0.2.0** — GitHub: per-user OAuth Device Flow, keychain storage, push the tag, create the GitHub Release
- **v0.3.0** — content: `relio post` templates, clipboard, publish hooks
- **v0.4.0** — plugin API (`BeforeRelease` / `AfterRelease` / `OnTagCreated` / `OnReleasePublished`)
- **v1.0.0** — `relio init` → `relio auth login` → `relio`, polished

---

## Contributing

See **[CONTRIBUTING.md](CONTRIBUTING.md)**. In short: Go 1.22+, `make test`
must stay green, `gofmt` everything, and commit with
[Conventional Commits](https://www.conventionalcommits.org/) — Relio dogfoods
its own release flow.

---

## License

MIT — see [LICENSE](LICENSE).
