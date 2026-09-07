<h1 align="center">Contributing to Relio</h1>

<p align="center"><i>from clone to pull request, step by step</i></p>

<p align="center">
  <img src="assets/divider.svg" alt="" width="100%">
</p>

Thanks for being here. Relio is a small, focused Go CLI, and the contributions
that keep it small and focused are the most welcome ones. You do **not** need to
be a Go expert — if you can read the code around your change and add a test for
it, you are in good shape.

## What you need first

| Tool | Why |
| ---- | --- |
| **Go 1.22+** | builds and runs Relio |
| **`git`** on your `PATH` | Relio and its tests call the real `git` binary |
| **`make`** *(optional)* | shortcuts around the `go` commands — everything works without it too |

<p align="center">
  <img src="assets/divider.svg" alt="" width="100%">
</p>

## Your first contribution in five steps

```bash
# 1. Get the code
git clone https://github.com/soyagvs/relio
cd relio

# 2. Make sure it's green before you touch anything
make test                 # go test ./...

# 3. Create a branch off main
git switch -c fix-the-thing

# 4. Make your change, then run it
make run ARGS="status"    # go run . status  — compiles fresh every time
make run ARGS="--yes"     # careful: this one actually creates a tag

# 5. Check it still passes, format, commit
make test
make tidy                 # gofmt -w . && go mod tidy
git commit -m "fix(status): show a hint when the tree is dirty"
```

Then open a pull request (see [below](#opening-a-pull-request)).

> [!TIP]
> `make run` recompiles from source on every call, so it is the fastest way to
> try a change. `make install` produces a real binary at `~/.cargo/bin/relio`
> that you can use in other repos — rerun it whenever you want those repos to
> see your latest change.

<p align="center">
  <img src="assets/divider.svg" alt="" width="100%">
</p>

## The four ground rules

1. **`make test` stays green.** Every change in behaviour comes with a test that
   covers it — new or updated.
2. **`gofmt` everything.** `make tidy` does it for you (`gofmt -w .` plus
   `go mod tidy`).
3. **`go vet ./...` is clean.** No new warnings.
4. **One concern per pull request.** Small and focused reviews faster and merges
   sooner.

<p align="center">
  <img src="assets/divider.svg" alt="" width="100%">
</p>

## Code conventions

- **Standard library first.** A new dependency needs a real reason. The image
  renderer's `fogleman/gg` + `mdp/qrterminal` are the current ceiling.
- **`internal/` packages stay decoupled.** The pure-logic packages
  (`conventional`, `semver`, `changelog`, `config`) must not import the TUI,
  `git`, or `cmd`. `cmd` is the layer that wires everything together.
- **Only `gitrepo` runs `git`.** It shells out to the binary — no CGO, no
  `go-git`. If you need a new git operation, add it there.
- **English everywhere in the code** — identifiers, comments, help text, UI
  strings, and generated output (changelog entries, card text) — no matter what
  language the discussion happens in.
- **Table-driven tests.** The Bubble Tea models (`menu`, `wizard`, `releases`,
  `pick`) are tested by feeding `tea.KeyMsg`s into `Update` and asserting on the
  resulting model or its `View()` — no real terminal involved. `card` is tested
  by rendering every shape/theme combination and checking the dimensions and
  that the bytes decode as a PNG.
- **The banner** (terminal ASCII art) lives in `internal/ui/ui.go` as `[]string`
  slices rendered with `lipgloss`; keep every row padded to the same width.

<p align="center">
  <img src="assets/divider.svg" alt="" width="100%">
</p>

## Commit messages

Relio uses **[Conventional Commits](https://www.conventionalcommits.org/)** and
runs its own release flow on itself, so your commit messages **directly drive**
the next version number and changelog. Get them right and the release paperwork
writes itself.

```
feat(image): add the --theme flag
fix(semver): treat a bare "v" prefix as valid
refactor(release): extract plan building from apply
docs: document relio status
chore: bump go directive to 1.22.2
```

| Prefix | Effect on the next release |
| ------ | -------------------------- |
| `feat:` | **MINOR** bump, shows under **Added** |
| `fix:` / `perf:` / `refactor:` | **PATCH** bump, shows under **Fixed** / **Changed** |
| `feat!:` or a `BREAKING CHANGE:` footer | **MAJOR** bump |
| `docs` / `chore` / `test` / `build` / `ci` | no bump, left out of the changelog |

> [!WARNING]
> **No AI or co-author attribution in commit messages.** Keep them to the
> Conventional Commits format above — nothing else.

<p align="center">
  <img src="assets/divider.svg" alt="" width="100%">
</p>

## Opening a pull request

1. Branch off `main`.
2. `make test` passes and `gofmt` is clean.
3. In the description, say **what** changed and **why**. Add screenshots for
   anything visual — the banner, `relio image` output, TUI screens.
4. Keep the diff reviewable — split unrelated changes into separate PRs.

<p align="center">
  <img src="assets/divider.svg" alt="" width="100%">
</p>

## Cutting a Relio release (maintainers only)

Relio releases itself:

```bash
relio status          # sanity-check what's unreleased
relio                 # or `relio --minor` / `relio --major` to force the bump
git push --follow-tags
```

Then CI takes over and attaches the cross-platform binaries to the GitHub
Release.
