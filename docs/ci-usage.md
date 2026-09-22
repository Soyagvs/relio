# Non-interactive / CI usage

Relio detects a non-TTY and behaves predictably — the menu and every prompt are
skipped, and a bare `relio` without `--yes` refuses to touch the repo.

```bash
relio --yes                          # infer + apply, no prompts
relio --minor --yes --no-changelog   # force minor, tag only
relio --yes --publish                # infer, tag, push, create the GitHub Release
relio --rc --yes                     # cut / advance a release candidate
relio check --strict                 # fail the job if any commit is not conventional
relio status                         # read-only, exit 0
relio post --format changelog        # plain text on stdout
relio image --shape horizontal       # latest tag, default theme, saved to cwd
```

`validate` / `before` / `after` hooks **do** run under `--yes` and in CI — that
is the point of them. `--publish` needs a `GITHUB_TOKEN` (or `GH_TOKEN`) in the environment;
with `--yes` it pushes and publishes without asking.

## Publishing Relio itself

Relio is distributed with **[GoReleaser](https://goreleaser.com)**. Cutting a
version is two steps:

1. **Locally** — bump + tag + changelog with Relio's own flow:

   ```bash
   relio            # or `relio --minor` / `relio --major`
   git push --follow-tags
   ```

2. **Automatically** — pushing a `vX.Y.Z` tag triggers
   `.github/workflows/release.yml`, which runs `goreleaser release`: it builds
   the platform binaries, packages `relio_<ver>_<os>_<arch>.tar.gz` (`.zip` on
   Windows), writes `checksums.txt`, creates the GitHub Release with every asset
   attached, and pushes an updated `Formula/relio.rb` to `Soyagvs/homebrew-tap`.

The config lives in [`.goreleaser.yaml`](../.goreleaser.yaml). It needs one secret
you set once: **`HOMEBREW_TAP_TOKEN`** — a PAT with write access to
`Soyagvs/homebrew-tap` (the repo's own `GITHUB_TOKEN` covers the Release itself).
