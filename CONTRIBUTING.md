# Contributing to Relio

Thanks for helping out. Relio is a small, focused Go CLI — contributions that
keep it that way are the most welcome.

## Prerequisites

- **Go 1.22+**
- The `git` binary on `PATH` (Relio and its tests shell out to it)
- `make` (optional, just wraps the `go` commands)

## Getting started

```bash
git clone https://github.com/soyagvs/relio
cd relio

make test                 # go test ./...
make run ARGS="status"    # go run . status
make run ARGS="--yes"     # go run . --yes  (careful — this creates a tag)
make build                # ./relio
make install              # build to ~/.cargo/bin/relio
```

`make run` compiles from source every time, so it's the fastest way to try a
change. `make install` produces a real binary you can use in other repos; rerun
it after any change you want reflected there.

## Ground rules

- **`make test` must stay green.** Add or update tests with every behavioural
  change.
- **`gofmt` everything** (`make tidy` runs `gofmt -w .` and `go mod tidy`).
- **`go vet ./...`** should be clean.
- Keep changes **small and focused** — one concern per PR.

## Code conventions

- **Standard library first.** New dependencies need a real reason; the image
  renderer's `fogleman/gg` + `mdp/qrterminal` are the current limit.
- **`internal/` packages stay decoupled.** Pure logic (`conventional`, `semver`,
  `changelog`, `config`) must not import the TUI, git, or `cmd`. `cmd` wires
  things together.
- **`gitrepo` is the only place that runs `git`.** It shells out to the binary
  — no CGO, no `go-git`.
- **English** for identifiers, comments, help text, UI strings, and generated
  artifacts (changelog entries, card text), regardless of the conversation
  language.
- **Table-driven tests.** The Bubble Tea models (`menu`, `wizard`, `releases`,
  `pick`) are tested by feeding `tea.KeyMsg`s to `Update` and asserting on the
  resulting model / `View()` — no real terminal needed. `card` is tested by
  rendering every shape/theme and checking dimensions + that it decodes as PNG.
- Terminal ASCII art (the banner) lives in `internal/ui/ui.go` as `[]string`
  slices rendered with `lipgloss`; keep rows padded to equal width.

## Commits

Relio uses **[Conventional Commits](https://www.conventionalcommits.org/)** and
dogfoods its own release flow, so commit messages directly drive its version and
changelog:

```
feat(image): add the --theme flag
fix(semver): treat a bare "v" prefix as valid
refactor(release): extract plan building from apply
docs: document relio status
chore: bump go directive to 1.22.2
```

- `feat:` → MINOR, `fix:` / `perf:` / `refactor:` → PATCH, `feat!:` or a
  `BREAKING CHANGE:` footer → MAJOR.
- `docs`, `chore`, `test`, `build`, `ci` do not appear in the changelog.
- **No AI / co-author attribution** in commit messages.

## Pull requests

1. Branch off `main`.
2. `make test` and `gofmt` clean.
3. Describe *what* changed and *why*. Screenshots for anything visual
   (banner, `relio image`, TUI screens).
4. Keep the diff reviewable — split unrelated changes.

## Cutting a Relio release (maintainers)

Relio releases itself:

```bash
relio status          # sanity-check what's unreleased
relio                  # or `relio --minor` / `relio --major` to force
git push --follow-tags
```

Then let CI attach the cross-platform binaries to the GitHub Release.
