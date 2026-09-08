<!-- Hero image goes here. Drop a file at docs/hero.png (≈720px wide). -->
<p align="center">
  <img width="2048" height="768" alt="Relio" src="https://github.com/user-attachments/assets/3a28d468-ab07-4024-8f39-a58c9b16ea42" />
</p>

<h1 align="center">Relio</h1>

<p align="center"><i>turn commits into releases</i></p>

<!--
  One badge identity: rounded "flat" pills, scaled up with height=28, charcoal
  label (labelColor=1c1c1c) + Relio orange (F5872B) across the board so the row
  reads as one Relio object. CI keeps shields' own pass/fail colour — a green
  check that can turn red is worth more than cohesion there.
-->
<p align="center">
  <a href="https://github.com/Soyagvs/relio/releases/latest"><img height="28" alt="latest release" src="https://img.shields.io/github/v/release/Soyagvs/relio?style=flat&labelColor=1c1c1c&color=F5872B&label=release"></a>
  <a href="https://github.com/Soyagvs/relio/releases"><img height="28" alt="downloads" src="https://img.shields.io/github/downloads/Soyagvs/relio/total?style=flat&labelColor=1c1c1c&color=F5872B&label=downloads"></a>
  <a href="https://github.com/Soyagvs/homebrew-tap"><img height="28" alt="homebrew tap" src="https://img.shields.io/badge/brew-soyagvs%2Ftap%2Frelio-F5872B?style=flat&labelColor=1c1c1c"></a>
  <br>
  <a href="https://github.com/Soyagvs/relio/actions/workflows/ci.yml"><img height="28" alt="ci status" src="https://img.shields.io/github/actions/workflow/status/Soyagvs/relio/ci.yml?branch=main&style=flat&labelColor=1c1c1c&label=ci"></a>
  <a href="https://github.com/Soyagvs/relio/stargazers"><img height="28" alt="stars" src="https://img.shields.io/github/stars/Soyagvs/relio?style=flat&labelColor=1c1c1c&color=F5872B"></a>
  <a href="LICENSE"><img height="28" alt="license" src="https://img.shields.io/github/license/Soyagvs/relio?style=flat&labelColor=1c1c1c&color=F5872B"></a>
</p>

<p align="center"><sub>badges are cached by shields.io — <code>relio stats</code> prints the live numbers</sub></p>

<p align="center">
  <img src="assets/divider.svg" alt="" width="100%">
</p>

## Relio in one minute

You just finished a feature. Now comes the boring part: figure out the new
version number, write the changelog, tag the commit, remember the exact `git`
commands. Every time. By hand.

**Relio does that part for you.** It reads the commits you already wrote, works
out what the next version should be, drafts the changelog, and creates the tag —
and it shows you a **preview first**, so nothing changes until you say yes.

```
code  →  commit  →  push  →  relio  →  version + CHANGELOG.md + tag
```

Think of it as the assistant that fills in the release paperwork. You still
review and sign it.

<table>
<tr>
<td width="33%" valign="top">

**What it reads**

Your git history — the commits since your last tag, written as
[Conventional Commits](https://www.conventionalcommits.org/)
(`feat:`, `fix:`, …).

</td>
<td width="33%" valign="top">

**What it gives back**

A version number, a `CHANGELOG.md` section, and an annotated git tag — all
previewed before anything is written.

</td>
<td width="33%" valign="top">

**What it never does**

Push to a remote, publish anything, or send data about you anywhere. Relio
runs locally and hands you the `git push` to run.

</td>
</tr>
</table>

> [!NOTE]
> Relio does **not** replace git or GitHub. It removes the repetitive work that
> happens *after* you finish coding.

<p align="center">
  <img src="assets/divider.svg" alt="" width="100%">
</p>

<h3 align="center">Download History</h3>

<p align="center">
  <img src="assets/download-history.svg" alt="Relio cumulative downloads over time" width="100%">
</p>

<p align="center"><sub>real snapshots, recorded daily into <a href=".github/stats/downloads.json"><code>.github/stats/downloads.json</code></a> — release-asset downloads, not unique installs</sub></p>

<div align="center">
<details>
<summary>Star history</summary>
<br>
<img src="assets/star-history.svg" alt="Relio stars over time" width="100%">
</details>
</div>

<p align="center">
  <img src="assets/divider.svg" alt="" width="100%">
</p>

## Contents

- [Install](#install)
- [Quick start](#quick-start)
- [The interactive menu](#the-interactive-menu)
- [Commands](#commands)
  - [`relio`](#relio--create-a-release) · [`relio status`](#relio-status) · [`relio check`](#relio-check) · [`relio stats`](#relio-stats) · [`relio init`](#relio-init) · [`relio post`](#relio-post) · [`relio image`](#relio-image) · [`relio auth`](#relio-auth) · [`relio version`](#relio-version)
- [Global flags](#global-flags)
- [How the version is chosen](#how-the-version-is-chosen)
- [How the changelog is built](#how-the-changelog-is-built)
- [Syncing the version into project files](#syncing-the-version-into-project-files)
- [Hooks](#hooks)
- [Configuration — `.release.yaml`](#configuration--releaseyaml)
- [Non-interactive / CI usage](#non-interactive--ci-usage)
- [Project layout](#project-layout)
- [Roadmap](#roadmap)
- [Contributing](#contributing)
- [License](#license)

<p align="center">
  <img src="assets/divider.svg" alt="" width="100%">
</p>

## Install

Pick whichever line matches how you already install tools. All of them give you
the same `relio` binary.

> Every version and its binaries: **[github.com/soyagvs/relio/releases](https://github.com/soyagvs/relio/releases)**

### Homebrew (macOS / Linux) — recommended

```bash
brew install soyagvs/tap/relio
```

`brew upgrade relio` picks up new releases. The formula downloads the official
prebuilt binaries from this repo's [GitHub Releases](https://github.com/soyagvs/relio/releases)
— it does not build from source.

### Manual — from GitHub Releases

Grab the archive for your platform from the
[latest release](https://github.com/soyagvs/relio/releases/latest), check it
against `checksums.txt`, and drop the binary on your `PATH`:

```bash
VER=1.2.0                      # the release you want
OS=darwin; ARCH=arm64          # darwin|linux|windows  +  amd64|arm64
curl -fsSLO "https://github.com/soyagvs/relio/releases/download/v${VER}/relio_${VER}_${OS}_${ARCH}.tar.gz"
curl -fsSLO "https://github.com/soyagvs/relio/releases/download/v${VER}/checksums.txt"
sha256sum -c --ignore-missing checksums.txt
tar -xzf "relio_${VER}_${OS}_${ARCH}.tar.gz" relio
sudo mv relio /usr/local/bin/
```

Windows: download `relio_<ver>_windows_amd64.zip` and put `relio.exe` on your
`PATH`.

### With Go

```bash
go install github.com/soyagvs/relio@latest   # -> $GOBIN / $GOPATH/bin
```

### From source

```bash
git clone https://github.com/soyagvs/relio
cd relio
go build -o relio .          # ./relio
# or: make install           # builds to ~/.cargo/bin/relio
```

Building requires **Go 1.22+**. At runtime Relio needs the `git` binary on
`PATH`.

<p align="center">
  <img src="assets/divider.svg" alt="" width="100%">
</p>

## Quick start

Three commands, run inside the project you want to release:

```bash
cd your-project
relio init        # writes .release.yaml (configuration only, never secrets)
relio             # opens the interactive menu
```

Pick **Create a release**, read the preview, confirm. Relio then:

1. updates `CHANGELOG.md`,
2. commits that one file as `chore(release): vX.Y.Z`,
3. creates an annotated git tag on that commit,
4. prints the exact `git push` for you to run.

> [!WARNING]
> **Nothing is pushed for you** by default. Relio stops at the tag and tells you
> the push command. You stay in control of what reaches the remote — opt in with
> `--publish` when you want Relio to push and create the GitHub Release too.

<p align="center">
  <img src="assets/divider.svg" alt="" width="100%">
</p>

## The interactive menu

Running `relio` with no arguments in a terminal opens a one-shot menu — it runs
**one action and exits**, so the result stays on screen. Run `relio` again for
another action.

```
  ██████╗ ███████╗██╗     ██╗  █████
  ██╔══██╗██╔════╝██║     ██║ █▪█ ███
  ██████╔╝█████╗  ██║     ██║████ ████
  ██╔══██╗██╔══╝  ██║     ██║████ ████
  ██║  ██║███████╗███████╗██║ ███ ███
  ╚═╝  ╚═╝╚══════╝╚══════╝╚═╝  █████
  turn commits into releases
  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  v1.1.0   ▲ v1.2.0 available
  created by SOYAGVS

▸ Status              What's unreleased since the last tag, and the suggested version
  Check               Lint the commits since the last tag — Conventional Commits and the bump
  Create a release    Version, changelog, and tag from commits since the last tag
  Releases            List every version, read its notes, or delete one
  Release text        Copy-paste announcement for social posts — pick a format
  Release image       Save a shareable PNG of a release — pick a shape
  GitHub auth         Token-based auth — check it with `relio auth status`
  Help                Every command and flag, with a one-line description
  Exit

↑/↓ move · enter select · q quit
```

In CI or when the output is piped (no TTY), or with `--yes` / a forced bump, the
menu is skipped and a release runs directly.

The banner shows the current version, and — when a newer Relio has been
published — `▲ vX.Y.Z available` right next to it. The check reads one cached
value on disk and, at most once a day, refreshes it in the background; it never
blocks the menu or sends anything about you. `relio version` shows the same hint.
Set `RELIO_NO_UPDATE_CHECK=1` (or run in CI) to turn it off.

<p align="center">
  <img src="assets/divider.svg" alt="" width="100%">
</p>

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
   ✓ CHANGELOG.md committed
   ✓ git tag v1.4.0 created
   ✓ release ready

     next:  git push && git push origin v1.4.0
   ```

`CHANGELOG.md` gets a new `## [x.y.z] - YYYY-MM-DD` section at the top of the
release history, in the [Keep a Changelog](https://keepachangelog.com/) format.
It is committed on its own as `chore(release): vX.Y.Z` (nothing else in the work
tree is touched), and the **annotated** git tag is created on that commit — so
the tag always carries its own changelog section. With `--no-tag` the changelog
is written but not committed, leaving the commit and tag to you.

**Publishing to GitHub** is opt-in. With `--publish` (or `github.release: true`
in `.release.yaml`), once the local tag exists Relio asks to push the branch and
tag to `origin` and create the GitHub Release, using the new changelog section
as the body. It needs a token — `GITHUB_TOKEN` or `gh auth login` (see
[`relio auth`](#relio-auth)). Without one, or if you decline the prompt, the
local release is untouched and Relio just prints the `git push` you can run
yourself.

| Flag                        | Meaning |
| --------------------------- | ------- |
| `--patch` `--minor` `--major` | Force the bump instead of inferring it from the commits (only one at a time). |
| `-y, --yes`                  | Skip the menu and the confirmation. Required in CI / a non-interactive shell. |
| `--no-changelog`             | Do not touch the changelog file. |
| `--no-tag`                   | Do not commit the changelog or create the git tag. |
| `--no-version-files`         | Do not update the files listed in `version_files`. |
| `--publish`                  | After tagging, push the branch and tag to `origin` and create the GitHub Release. |
| `--rc`                       | Cut a release candidate (`vX.Y.Z-rc.N`) instead of the final version. See [Pre-releases](#pre-releases). |
| `--no-hooks`                 | Skip the `before` / `after` hooks from `.release.yaml` for this run. See [Hooks](#hooks). |

```bash
relio                 # interactive
relio --yes           # apply the inferred bump, no prompts
relio --minor --yes   # force a minor bump
relio --no-tag        # write the changelog only
relio --publish       # also push and create the GitHub Release
relio --rc            # cut the next release candidate
relio --no-hooks      # skip the .release.yaml hooks for this run
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

When the current version is a pre-release, `status` adds one hint line under the
`Suggested` row: `` on a pre-release — `relio` finalizes v1.6.0, `relio --rc`
cuts the next rc ``.

---

### `relio check`

Lints the commits since the last tag: how many are
[Conventional Commits](https://www.conventionalcommits.org/), which are not, and
the bump they add up to. Read-only — no prompts, no writes, safe anywhere.

```
$ relio check

azeink

8 commits since v1.5.0
  ✓ 5 conventional
  ✗ 3 not conventional:
      a1b2c3d  add login screen
      d4e5f6a  dashboard fix
      f7g8h9i  wip

Detected bump: minor  →  v1.6.0
```

When every commit is conventional the `✗` block is dropped. With no commits since
the tag it prints `Nothing to check …` and exits 0.

| Flag       | Meaning |
| ---------- | ------- |
| `--strict` | Exit non-zero when at least one commit is not a Conventional Commit. Handy in a pre-release CI gate. |

Also available as the **Check** menu entry.

---

### `relio stats`

Relio's **public** distribution numbers, read straight from the GitHub REST API
— release-asset download counts, per-release and per-platform breakdowns, stars
and forks.

```
$ relio stats

Relio -- stats

Downloads
  total               1,284
  latest release        437

Releases
  v1.2.0                437
  v1.1.0                521
  v1.0.0                326

Latest release (v1.2.0)
  macOS arm64           291
  macOS amd64            38
  Linux amd64            82
  Linux arm64            26

GitHub
  stars                126
  forks                 14

downloads = release-asset downloads, not unique users or installs
```

- **No telemetry.** Relio records nothing — no runs, commands, repos, or user
  data. `relio stats` only makes `GET` requests to `api.github.com`.
- **"Downloads" = release-asset downloads.** Each time someone downloads a
  binary archive from a GitHub Release it counts once. It is **not** a count of
  unique users or active installs. `checksums.txt` and signatures are excluded.
- Works **without authentication**. If you hit the API rate limit, set
  `GITHUB_TOKEN` (or `GH_TOKEN`) in your environment to raise it — the token is
  only sent to GitHub, never stored.
- `--repo owner/name` queries a different repository (defaults to
  `soyagvs/relio`). `--prerelease` includes pre-releases in the list.

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

Renders a dark, developer-styled **release card**. Everything on it comes from
the real release. In a terminal it prompts for the release, the shape, the
colour, and then **what to do with the image**:

- **Save + download link** — write `relio-<version>-<shape>.png` to the current
  directory *and* upload it for a link + QR.
- **Save only** — just write the file.
- **Download link only** — upload it for a link + QR, write nothing to disk.

The flags below skip those prompts. With no TTY and no flags it just saves the
PNG to the current directory (latest tag / horizontal / orange).

| Flag        | Values |
| ----------- | ------ |
| `--version` | Release tag to render. Default: the latest tag. |
| `--shape`   | `horizontal` (1200×630, Twitter/OG) · `vertical` (1080×1920, Instagram story) · `square` (1080×1080). Each has its **own** responsive layout, not a crop. |
| `--theme`   | Accent colour: `orange` *(default)* · `green` · `purple`. The section colours (Added green / Changed amber / Fixed coral) are fixed. |
| `--hash`    | Prefix each line with its short commit hash. Off by default. |
| `--upload`  | Save the PNG **and** upload it to a temporary public host (litterbox, 72h; catbox as fallback), printing the URL **plus a QR code**. Handy for getting it onto a phone over `mosh` — only text crosses the wire. |
| `--link-only` | Upload for the URL + QR **without** writing a file to disk. |

```bash
relio image                                   # prompts for everything
relio image --shape square --theme green
relio image --version v1.4.0 --shape vertical --upload
relio image --link-only                        # just give me the link
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

Relio talks to GitHub with a **personal access token**, not a login of its own.
It checks `RELIO_GITHUB_TOKEN`, `GITHUB_TOKEN` and `GH_TOKEN` in that order, then
falls back to `gh auth token` when the [GitHub CLI](https://cli.github.com/) is
signed in. The token is only ever sent to `api.github.com` in the
`Authorization` header — nothing is written to disk.

```
$ relio auth status
· logged in as octocat (via GITHUB_TOKEN)
```

`relio auth status` resolves the token and prints who it belongs to (or
`not authenticated` when none is found). `login` / `logout` are short notes: set
`GITHUB_TOKEN` to a PAT with `repo` scope — or run `gh auth login` — and unset
those vars (or `gh auth logout`) to drop it. A device-flow login with keychain
storage is still planned.

---

### `relio version`

```
$ relio version
Relio v1.4.0 (commit a1b2c3d, built 2026-09-06)
```

The version is `dev` unless the binary was built with `HEAD` exactly on a tag
(that's what `make install` does), or with
`-ldflags "-X github.com/soyagvs/relio/cmd.version=…"`.

<p align="center">
  <img src="assets/divider.svg" alt="" width="100%">
</p>

## Global flags

Available on every command:

| Flag              | Meaning |
| ----------------- | ------- |
| `-C, --dir <path>` | Run as if Relio was started in `<path>`. |
| `--no-hash`         | Hide the commit hash on each release-note line (preview, Releases browser, `post`, `check`). |

<p align="center">
  <img src="assets/divider.svg" alt="" width="100%">
</p>

## How the version is chosen

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
| only `fix:` / `perf:` / `refactor:` | **PATCH** | `1.3.2 → 1.3.3` |
| commits, but no conventional signal | **PATCH** | `1.3.2 → 1.3.3` |
| no commits since the last tag | — | *nothing to release* |

`--patch` / `--minor` / `--major` override the inference; the wizard's
"Change to…" options do the same interactively.

> [!TIP]
> Pre-1.0 (`0.x.y`): the API is not stable yet, so breaking changes inside a
> MINOR are conventionally acceptable — Relio still bumps MAJOR if you ask it to.

### Pre-releases

`relio --rc` cuts a **release candidate** — a `vX.Y.Z-rc.N` tag — instead of the
final version. The typical flow:

1. On stable `v1.5.0` with a `feat:` since, `relio --rc` cuts `v1.6.0-rc.1`.
2. More commits land — `relio --rc` cuts `v1.6.0-rc.2` (same core, counter up).
3. Ready to ship — `relio` (no flag) finalizes `v1.6.0`.
4. If a commit since the last rc escalates the target (a `feat!`, say),
   `relio --rc` moves the core up and restarts the counter: `v2.0.0-rc.1`.

The pre-release value carries all the way through: the git tag, the changelog
section heading, and any `version_files`. Finalizing summarises the **whole
span** since the last stable tag, so the `v1.6.0` section covers every commit
made across `rc.1`, `rc.2`, and anything after.

GoReleaser already treats a tag with a `-` as a pre-release: `.goreleaser.yaml`
carries `prerelease: auto` (marks the GitHub Release as a pre-release) and
`skip_upload: auto` (skips the Homebrew tap bump) — no config change needed.

`relio --rc` still opens the interactive menu in a TTY; it does not imply
`--yes`.

<p align="center">
  <img src="assets/divider.svg" alt="" width="100%">
</p>

## How the changelog is built

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

<p align="center">
  <img src="assets/divider.svg" alt="" width="100%">
</p>

## Syncing the version into project files

By default `relio` writes the next version only into the git tag and the
changelog. Opt in to keeping other files in sync by listing them under
`release.version_files` — each one gets the new version written into it **in the
same `chore(release): vX.Y.Z` commit** as the changelog, so the tag points at a
commit where every file agrees on the version.

```yaml
release:
    version_files:
        - package.json                       # known filename — built-in rule
        - VERSION                             # whole-file version token
        - Cargo.toml                          # first `version = "…"` line
        - { path: src/app/__init__.py, pattern: '__version__ = "([^"]+)"' }
```

Known filenames need no pattern: `package.json`, any `*.toml` (`Cargo.toml`,
`pyproject.toml`, …), and `VERSION` / `version.txt`. For anything else give a
`{ path, pattern }` entry whose `pattern` is a Go regexp with **exactly one
capture group** wrapping the version substring.

Replacement is regex-based — Relio never reformats the file, it swaps the matched
span and leaves every other byte (indentation, key order, trailing newline)
untouched. The value written is the **bare number**, no leading `v`
(`1.6.0`). A re-run when a file is already at the target version is a no-op.
Pass `--no-version-files` to skip the whole step for one run.

<p align="center">
  <img src="assets/divider.svg" alt="" width="100%">
</p>

## Hooks

Run your own shell commands around a release by listing them under
`release.hooks`:

```yaml
release:
    hooks:
        before: make test                       # one command…
        after:                                   # …or a list, run in order
            - ./scripts/changelog-to-slack.sh
            - echo done
```

Each hook is a single shell command string, or a list of them. A list stops at
the first command that exits non-zero.

- **`before`** runs *after* you confirm the release and *before* anything is
  written. A non-zero exit **aborts** the release — nothing is written, no tag —
  and `relio` exits non-zero.
- **`after`** runs at the very end: after the tag, and after the `--publish`
  push + GitHub Release when that ran. A non-zero exit prints a **warning** but
  `relio` still exits 0 — the release is already done.

Hooks run with the **repository root** as the working directory, inherit the
process environment and stdout/stderr, and get three extra variables:

| Variable | Example | Meaning |
| -------- | ------- | ------- |
| `RELIO_VERSION` | `1.6.0` | the new version, no `v` |
| `RELIO_TAG` | `v1.6.0` | the new tag, honouring `tag_prefix` |
| `RELIO_PREVIOUS_TAG` | `v1.5.0` | the tag the release was computed from (empty on a first release) |

Pass `--no-hooks` to skip every hook for one run. `--yes` does **not** skip
hooks — CI needs them to run. Hooks never run for `relio status` or
`relio check` (they don't apply anything).

<p align="center">
  <img src="assets/divider.svg" alt="" width="100%">
</p>

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
    version_files: []            # files to sync to the new version, e.g.
                                 #   - package.json
                                 #   - { path: foo.py, pattern: '__version__ = "([^"]+)"' }
    hooks:                       # shell commands run around a release (see Hooks)
        before: ""               #   string or list — non-zero exit aborts the release
        after: []                #   string or list — non-zero exit only warns

github:
    enabled: false         # reserved
    repo: ""               # "owner/name" override; empty = derive from the origin remote
    release: false         # on `relio`, also push and create the GitHub Release (same as --publish)

content:
    enabled: false         # reserved for v0.3.0
```

Missing fields fall back to these defaults, so a minimal file with just
`project:` works. Unknown `versioning` / `commits` values are rejected with a
clear error.

<p align="center">
  <img src="assets/divider.svg" alt="" width="100%">
</p>

## Non-interactive / CI usage

Relio detects a non-TTY and behaves predictably:

```bash
relio --yes                          # infer + apply, no prompts (fails without --yes)
relio --minor --yes --no-changelog   # force minor, tag only
relio status                         # read-only, exit 0
relio post --format changelog        # plain text on stdout
relio image --shape horizontal       # latest tag, default theme, saved to cwd
```

By default Relio never pushes — wire the `git push` into your pipeline. Pass
`--publish` with `--yes` and a `GITHUB_TOKEN` in the environment to have Relio
push the branch and tag and create the GitHub Release itself, no prompt.

### Publishing Relio itself

Relio is distributed with **[GoReleaser](https://goreleaser.com)**. Cutting a
version is two steps:

1. **Locally** — bump + tag + changelog with Relio's own flow:

   ```bash
   relio            # or `relio --minor` / `relio --major`
   git push --follow-tags
   ```

2. **Automatically** — pushing a `vX.Y.Z` tag triggers
   `.github/workflows/release.yml`, which runs `goreleaser release`:
   builds the 5 platform binaries, packages `relio_<ver>_<os>_<arch>.tar.gz`
   (`.zip` on Windows), writes `checksums.txt`, creates the GitHub Release with
   every asset attached, and pushes an updated `Formula/relio.rb` to
   `Soyagvs/homebrew-tap`.

The config lives in [`.goreleaser.yaml`](.goreleaser.yaml). It needs one secret
you set once: **`HOMEBREW_TAP_TOKEN`** — a PAT with write access to
`Soyagvs/homebrew-tap` (the repo's own `GITHUB_TOKEN` covers the Release
itself).

<p align="center">
  <img src="assets/divider.svg" alt="" width="100%">
</p>

## Project layout

```
main.go
.goreleaser.yaml     GoReleaser: build matrix, archives, checksums, GitHub Release, Homebrew tap
.github/
  workflows/ci.yml       vet · gofmt · test -race
  workflows/release.yml   tag push -> GoReleaser
  workflows/stats.yml     daily -> record a download/star snapshot, rebuild the SVGs
  stats/downloads.json    the recorded history (one {date,total,stars} row per day)
assets/
  divider.svg            the orange rule used across this README
  download-history.svg    generated from stats/downloads.json — never hand-edited
  star-history.svg        generated the same way
cmd/                 Cobra command wiring (root, status, stats, init, post, image, auth, version)
tools/
  statsnap/          one-shot job behind stats.yml: fetch counts, upsert a row, redraw the SVGs
internal/
  conventional/      Conventional Commits parser
  semver/            version parsing + bump rules
  changelog/         Keep a Changelog rendering, section extract/remove
  config/            .release.yaml load / save
  gitrepo/           thin wrapper over the git binary
  release/           orchestration: build a plan, apply it
  ghstats/           read-only GitHub REST client for `relio stats`
  update/            best-effort "newer relio available" check for the menu (cached, non-blocking)
  statchart/         tiny dependency-free line-chart -> SVG string
  ui/                lipgloss palette, banner, non-interactive views
  menu/              Bubble Tea main menu
  wizard/            Bubble Tea release confirmation
  releases/          Bubble Tea release browser (view / delete)
  pick/              reusable Bubble Tea single-select
  card/              PNG release-card renderer (fogleman/gg + Go fonts)
  upload/            temporary file host client (litterbox / catbox)
```

`gitrepo` shells out to the system `git` — no CGO, no `go-git`.

<p align="center">
  <img src="assets/divider.svg" alt="" width="100%">
</p>

## Roadmap

- **v0.1.x** — local git tool: parse, version, changelog, tag, preview · status · post · image *(current)*
- **v0.2.0** — GitHub: token-based Release creation — push the branch and tag, create the GitHub Release from the changelog *(done)*; per-user OAuth Device Flow + keychain storage still to come
- **v0.3.0** — content: `relio post` templates, clipboard, publish hooks
- **v0.4.0** — plugin API (`BeforeRelease` / `AfterRelease` / `OnTagCreated` / `OnReleasePublished`)
- **v1.0.0** — `relio init` → `relio auth login` → `relio`, polished

<p align="center">
  <img src="assets/divider.svg" alt="" width="100%">
</p>

## Contributing

New here? See **[CONTRIBUTING.md](CONTRIBUTING.md)** — it walks you from clone to
pull request. The short version: Go 1.22+, `make test` stays green, `gofmt`
everything, and commit with
[Conventional Commits](https://www.conventionalcommits.org/) — Relio dogfoods
its own release flow.

<p align="center">
  <img src="assets/divider.svg" alt="" width="100%">
</p>

## License

MIT — see [LICENSE](LICENSE).
