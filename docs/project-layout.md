# Project layout

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
cmd/                 Cobra command wiring (root, status, check, guide, stats, init, post, image, auth, version)
tools/
  statsnap/          one-shot job behind stats.yml: fetch counts, upsert a row, redraw the SVGs
internal/
  i18n/              UI translation catalog (English / Español) + active-language resolution
  userconfig/        global config.yaml (per-OS user config dir) — currently just the language preference
  settings/          Bubble Tea Settings screen (language, release-footer toggles)
  conventional/      Conventional Commits parser
  semver/            version parsing + bump rules + pre-release counters
  changelog/         Keep a Changelog rendering, section extract/remove
  versionfile/       regex-based version sync for package.json / *.toml / VERSION / custom patterns
  hook/              validate/before/after shell hooks (sh -c / cmd /c), streamed, RELIO_* env
  config/            .release.yaml load / save / in-place field edits (SetFields)
  gitrepo/           thin wrapper over the git binary
  release/           orchestration: build a plan, apply it
  ghstats/           read-only GitHub REST client for `relio stats`
  ghrelease/         write-side GitHub client: token resolution + create a Release for `--publish`
  guide/             `relio guide` walkthrough (interactive stepper / plain text)
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
