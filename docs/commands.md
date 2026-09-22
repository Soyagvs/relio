# Commands

## `relio` — create a release

With no subcommand, `relio` builds a **release plan** from the commits since the
last tag and applies it.

1. Find the latest tag (`git describe --tags --abbrev=0`); if there is none,
   start from `v0.0.0`.
2. Read `<tag>..HEAD`, parse each commit as a Conventional Commit.
3. Decide the next version (see [How the version is chosen](versioning.md)).
4. Show the preview — **nothing is written yet**:

   ```
   ⬢ Relio  v1.5.0  ·  azeink

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
It is committed on its own as `chore(release): vX.Y.Z` — together with any
[version files](version-sync.md) you configured, and
nothing else in the work tree — and the **annotated** git tag is created on that
commit, so the tag always carries its own changelog section. With `--no-tag` the
changelog is written but not committed, leaving the commit and tag to you.

Each section can also carry a `Thanks to …` credits line and a GitHub compare
link — both off by default. While they are, `relio` prints a one-line reminder
after the release; see [Contributors and a compare link](changelog-generation.md#contributors-and-a-compare-link)
to turn them on.

| Flag | Meaning |
| ---- | ------- |
| `--patch` `--minor` `--major` | Force the bump instead of inferring it from the commits (one at a time). |
| `-y, --yes` | Skip the menu and the confirmation. Required in CI / a non-interactive shell. |
| `--no-changelog` | Do not touch the changelog file. |
| `--no-tag` | Do not commit the release files or create the git tag. |
| `--no-version-files` | Skip the `release.version_files` sync for this run. See [Syncing the version into project files](version-sync.md). |
| `--publish` | After tagging, push the branch and tag to `origin` and create the GitHub Release. See [Publishing to GitHub](publishing.md). |
| `--rc` | Cut a release candidate (`vX.Y.Z-rc.N`) instead of the final version. See [Pre-releases](versioning.md#pre-releases). |
| `--no-hooks` | Skip the `validate` / `before` / `after` hooks from `.release.yaml` for this run. See [Release hooks](release-hooks.md). |
| `--edit` | Open the generated release notes in your editor before anything is written. Interactive terminals only. |

`--edit` opens `$RELIO_EDITOR` / `$VISUAL` / `$EDITOR` (falling back to `vi`) on
the generated notes body once you have confirmed the plan. Save your version to
use it, or save an empty or unchanged buffer to keep the generated notes. The
edited text lands verbatim in both `CHANGELOG.md` and the GitHub Release body.
The `## [x.y.z]` heading is not editable — only the body.

```bash
relio                     # interactive
relio --yes               # apply the inferred bump, no prompts
relio --minor --yes       # force a minor bump
relio --no-tag            # write the changelog only
relio --yes --publish     # also push and create the GitHub Release
relio --rc --yes          # cut the next release candidate
relio --yes --no-hooks    # skip the .release.yaml hooks for this run
relio --edit --yes        # confirm nothing, but hand-edit the notes
```

---

## `relio status`

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
`Suggested` row:

```
on a pre-release — `relio` finalizes v1.6.0, `relio --rc` cuts the next rc
```

---

## `relio check`

Read-only lint of the commits since the last tag: how many are
[Conventional Commits](https://www.conventionalcommits.org/), which are not, and
the bump they add up to. No prompts, no writes, safe anywhere.

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

| Flag | Meaning |
| ---- | ------- |
| `--strict` | Exit non-zero when at least one commit since the tag is not a Conventional Commit. Use it as a pre-release CI gate. |

Also available as the **Check** menu entry.

---

## `relio undo`

Reverse the release you just cut — the tag, and the `chore(release):` commit that
carries the changelog and any version-file bumps. It only works while the release
is still local: it never touches a remote, and it is not the way to walk back
something you have already pushed.

```
$ relio undo

Undo v1.6.0
  · delete the local tag v1.6.0
  · remove the `chore(release): v1.6.0` commit (git reset --hard HEAD~1)
    CHANGELOG.md and any version files return to their previous state

  Proceed? [y/N] y
✓ deleted tag v1.6.0
✓ removed the release commit
```

With no tags yet it prints `No tags yet …` and exits 0. It refuses in two cases:
when the latest tag no longer points at `HEAD` (you have committed since the
release, so there is nothing safe to unwind), and when `HEAD` is already on a
remote branch — undoing then would rewrite shared history, so it tells you to
delete the tag on the remote yourself and drop the GitHub Release by hand.

| Flag | Meaning |
| ---- | ------- |
| `-y`, `--yes` | Skip the confirmation prompt. |
| `--force` | Undo even when the working tree is dirty. `git reset --hard` discards those uncommitted changes, so use it deliberately. |

`--no-tag` runs and GitHub Releases are out of scope: if you released without a
tag, or want a published Release gone, do that step by hand.

---

## `relio releases edit`

Fix a typo (or anything else) in a `CHANGELOG.md` section that has already
been published — without reinventing the extraction/removal logic Relio
already uses internally.

```
$ relio releases edit 1.4.0
```

This opens just that version's section in your editor (`$RELIO_EDITOR`,
`$VISUAL`, `$EDITOR`, in that order, falling back to `vi`/`notepad`). Save and
close to apply the edit, or leave the text unchanged to cancel:

- If nothing changed, it prints `Unchanged.` and leaves the file alone.
- If you clear the section entirely, it refuses to write an empty section.
- Otherwise it rewrites `CHANGELOG.md` with your edited section swapped in,
  in place — every other section is untouched.

It **only rewrites the changelog file** — no commit, no tag, no git operation
of any kind. Review the diff and commit it yourself when you're happy:

```bash
git add CHANGELOG.md && git commit
```

---

## `relio guide`

A step-by-step walkthrough of the whole flow, for when you are meeting Relio for
the first time: `relio init` → writing Conventional Commits → `relio status` →
`relio` → pushing or `--publish` → the optional extras (`post`, `image`,
`version_files`, hooks).

In a terminal it is an interactive stepper — `enter` to move on, `←` to go back,
`q` to leave — and on the steps where it helps it offers to **run the command
for you**: `relio init` when there is no `.release.yaml` yet, and `relio check` /
`relio status` once there is one. Piped or redirected, it prints the same eight
steps as plain text.

```bash
relio guide          # interactive in a TTY, plain text when piped
```

Also reachable as the **Guide** menu entry, or by pressing `g` anywhere in the
menu.

---

## `relio stats`

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
  unique users or active installs. `checksums.txt` is excluded.
- Works **without authentication**. If you hit the API rate limit, set
  `GITHUB_TOKEN` (or `GH_TOKEN`) in your environment to raise it — the token is
  only sent to GitHub, never stored.
- `--repo owner/name` queries a different repository (defaults to
  `soyagvs/relio`). `--prerelease` includes pre-releases in the list.

---

## `relio init`

Writes `.release.yaml` at the repository root with sensible defaults. The file
holds **configuration only** — credentials never go in it. Refuses to overwrite
an existing file.

| Flag | Meaning |
| ---- | ------- |
| `--project` | Project name. Defaults to the `origin` remote's repo name, else the directory name. |

```bash
relio init
relio init --project azeink
```

Also the **Setup** menu entry, which prompts for the project name.

---

## `relio post`

Generates a short, plain-text announcement from the commits since the last tag.
**Only the text goes to stdout**, so `relio post | pbcopy` (or `| wl-copy`)
copies it cleanly. Colour is added when stdout is a terminal and stripped when
it is piped.

| `--format` | Output |
| ---------- | ------ |
| `minimal` *(default)* | Byte-identical to what the **Releases** browser prints for a version: `relio -- release`, `<project> · <version>`, `<date> · <time> · N commits`, then the grouped notes. |
| `social` | Shortest. `Project -- Release`, then `version · DD.MM.YY · HH:MM`, then one `<type>  <description>` line per **notable** commit (`feat`, `fix`, `perf`, `refactor`, `revert`, `style`). |
| `technical` | Terse `•` bullet list — good for a changelog or a dev channel. |
| `casual` | Loose tone: `proj v1.4.0 is out. → …` |
| `changelog` | The exact section that goes into `CHANGELOG.md`. |

```
$ relio post --format social
Azeink -- Release

v1.4.0 · 06.09.26 · 14:36

feat      Add facial attendance
feat      Add new kiosk interface
refactor  Authentication flow
fix       Supervisor login
```

*(This is a preview of the content generator — nothing is published.)* Also the
**Announcement** menu entry.

---

## `relio image`

Renders a dark, developer-styled **release card**. Everything on it comes from
the real release. In a terminal it prompts for the release, the shape, the
colour, and then **what to do with the image**:

- **Save + download link** — write `relio-<version>-<shape>.png` to the current
  directory *and* upload it for a link + QR.
- **Save only** — just write the file.
- **Download link only** — upload it for a link + QR, write nothing to disk.

The flags below skip those prompts. With no TTY and no flags it just saves the
PNG to the current directory (latest tag / horizontal / orange).

| Flag | Values |
| ---- | ------ |
| `--version` | Release tag to render. Default: the latest tag. |
| `--shape` | `horizontal` (1200×630, Twitter/OG) · `vertical` (1080×1920, Instagram story) · `square` (1080×1080). Each has its **own** responsive layout, not a crop. |
| `--theme` | Accent colour: `orange` *(default)* · `green` · `purple`. The section colours (Added / Changed / Fixed) are fixed. |
| `--hash` | Prefix each line with its short commit hash. Off by default. |
| `--upload` | Save the PNG **and** upload it to a temporary public host (litterbox, 72h; catbox as fallback), printing the URL **plus a QR code**. Handy for getting it onto a phone over `mosh` — only text crosses the wire. |
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

## `relio auth`

Relio talks to GitHub with a **personal access token you already have**, not a
login of its own. It checks `RELIO_GITHUB_TOKEN`, `GITHUB_TOKEN` and `GH_TOKEN`
in that order, then falls back to `gh auth token` when the
[GitHub CLI](https://cli.github.com/) is signed in. The token is only ever sent
to `api.github.com` in the `Authorization` header — nothing is written to disk.

```
$ relio auth status
· logged in as octocat (via GITHUB_TOKEN)
```

`relio auth status` resolves the token, calls `GET /user`, and prints who it
belongs to and which source it came from — or `not authenticated` when none is
found. It is the same check as the **Auth** menu entry.

There is no `login` or `logout` of Relio's own. `relio auth login` and
`relio auth logout` just print the one-liners:

- **connect** — set `GITHUB_TOKEN` (or `RELIO_GITHUB_TOKEN` / `GH_TOKEN`) to a
  PAT with `repo` scope, or run `gh auth login`.
- **disconnect** — unset those variables, or run `gh auth logout`.

A per-user browser sign-in (OAuth Device Flow) with OS-keychain storage is on
the [roadmap](../README.md#roadmap).

---

## `relio version`

```
$ relio version
Relio v1.5.0 (commit a1b2c3d, built 2026-09-07)
```

The version is `dev` unless the binary was built with `HEAD` exactly on a tag
(that's what `make install` does), or with
`-ldflags "-X github.com/soyagvs/relio/cmd.version=…"` (that's what the official
release build does). It also carries the same `▲ vX.Y.Z available` hint as the
menu.

## Global flags

Available on every command:

| Flag | Meaning |
| ---- | ------- |
| `-C, --dir <path>` | Run as if Relio was started in `<path>`. |
| `--no-hash` | Hide the commit hash on each release-note line (preview, Releases browser, `post`, `check`). |
