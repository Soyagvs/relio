<!-- Hero image goes here. Drop a file at docs/hero.png (≈720px wide). -->
<p align="center">
  <img width="2056" height="765" alt="ChatGPT Image 15 sept 2026, 19_22_46" src="https://github.com/user-attachments/assets/fddec483-221b-4937-8107-7244bf694b43" />
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

**What leaves your machine**

Nothing, by default. Relio stops at the local tag and prints the `git push`.
Pushing and creating the GitHub Release is opt-in — `--publish`.

</td>
</tr>
</table>

> [!NOTE]
> Relio does **not** replace git or GitHub. It removes the repetitive work that
> happens *after* you finish coding, and it never sends data about you anywhere.

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

## Documentation

**Getting started**

| Doc | What's in it |
| --- | ------------ |
| [Install](docs/install.md) | Homebrew, manual download, `go install`, from source, shell completion |
| [Quick start](#quick-start) | The fastest path to your first release (below) |
| [The interactive menu](docs/interactive-menu.md) | What `relio` with no arguments does |

**Commands & usage**

| Doc | What's in it |
| --- | ------------ |
| [Commands](docs/commands.md) | Every subcommand (`status`, `check`, `undo`, `guide`, `stats`, `init`, `post`, `image`, `auth`, `version`, …) and the global flags |
| [Non-interactive / CI usage](docs/ci-usage.md) | Running Relio in scripts and CI, and publishing Relio itself |

**How it works**

| Doc | What's in it |
| --- | ------------ |
| [How the version is chosen](docs/versioning.md) | SemVer inference and pre-releases (`--rc`) |
| [How the changelog is built](docs/changelog-generation.md) | Commit-type mapping, contributors line, compare link |
| [Publishing to GitHub](docs/publishing.md) | `--publish`, tokens, and the GitHub Release flow |
| [Syncing the version into project files](docs/version-sync.md) | Keeping `package.json`, `VERSION`, etc. in step |
| [Release hooks](docs/release-hooks.md) | `validate` / `before` / `after` shell hooks |

**Configuration**

| Doc | What's in it |
| --- | ------------ |
| [Configuration — `.release.yaml`](docs/configuration.md) | Every field, with defaults |
| [Language](docs/language.md) | English / Español, and how it's resolved |

**Contributing**

| Doc | What's in it |
| --- | ------------ |
| [Project layout](docs/project-layout.md) | Map of the codebase |
| [Contributing](#contributing) | How to submit a change (below) |

<p align="center">
  <img src="assets/divider.svg" alt="" width="100%">
</p>

## Quick start

Two commands, run inside the project you want to release:

```bash
cd your-project
relio init        # writes .release.yaml (configuration only, never secrets)
relio             # opens the interactive menu
```

Pick **Release**, read the preview, confirm. Relio then:

1. updates `CHANGELOG.md`,
2. commits that file as `chore(release): vX.Y.Z`,
3. creates an annotated git tag on that commit,
4. prints the exact `git push` for you to run.

> [!WARNING]
> **Nothing is pushed for you** by default. Relio stops at the tag and tells you
> the push command. You stay in control of what reaches the remote — opt in with
> `--publish` (or the menu's Release entry) when you want Relio to push and
> create the GitHub Release too.

<p align="center">
  <img src="assets/divider.svg" alt="" width="100%">
</p>

## Roadmap

- **Shipped** — commit parsing, SemVer inference, changelog, annotated tags,
  preview + wizard · `relio status` · `relio check` · `relio undo` ·
  `relio releases edit` · `relio guide` · release candidates (`--rc`) ·
  `version_files` sync · `validate` / `before` / `after` hooks ·
  `relio post` / `relio image` · token-based GitHub Release publishing
  (`--publish`) · hand-editing the notes before writing (`--edit`) · changelog
  footer — contributors line + compare link.
- **Next** — per-user GitHub sign-in via OAuth Device Flow with OS-keychain
  storage, so `--publish` no longer needs a token you supplied yourself.
- **Later** — release plugins (`BeforeRelease` / `AfterRelease` in Go), richer
  `relio post` templates.

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
