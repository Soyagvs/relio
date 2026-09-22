# Publishing to GitHub

By default Relio stops at the local tag. Getting that release onto GitHub then
means `git push`, a trip to the Releases page, and pasting the notes in by hand.
`--publish` does all of it in the same run.

## How it works

Turn it on per run with `--publish`, or always with `github.release: true` in
`.release.yaml`. Once the local tag exists, Relio:

1. pushes the current branch to `origin`,
2. pushes the new tag to `origin`,
3. creates a GitHub Release for that tag, using the new changelog section (its
   `## […]` heading stripped) as the body. A `-` in the tag marks the Release as
   a **pre-release**.

The target repo comes from `github.repo` (`owner/name`) when you set it,
otherwise it is read from the `origin` remote — which must be a `github.com`
remote.

**Auth is a token you already have.** Relio checks, in order:
`RELIO_GITHUB_TOKEN`, `GITHUB_TOKEN`, `GH_TOKEN`, then `gh auth token` when the
GitHub CLI is signed in. The token needs `repo` scope. There is **no OAuth app,
no browser flow, and nothing stored by Relio** — the token only travels to
`api.github.com` in the `Authorization` header.

In a terminal Relio asks before it pushes:

```
Push main and v1.6.0 to origin and publish the GitHub Release? [y/N]
```

With `--yes` it does not ask. If there is **no token**, or you **decline** the
prompt, Relio prints the `git push` commands and stops — the local tag is
untouched. If a Release **already exists** for the tag, Relio leaves it alone and
says so.

## Example

```bash
export GITHUB_TOKEN=ghp_xxxxxxxx     # or: gh auth login
relio --yes --publish
```

```
✓ CHANGELOG.md updated
✓ CHANGELOG.md committed
✓ git tag v1.6.0 created
✓ release ready

✓ pushed to origin
✓ GitHub Release v1.6.0 published
  https://github.com/you/project/releases/tag/v1.6.0
```

> [!NOTE]
> **Maintaining Relio itself?** GoReleaser in CI still does the heavier job on a
> tag push — cross-platform binaries, `checksums.txt`, the Homebrew tap bump (see
> [Non-interactive / CI usage](ci-usage.md)). `--publish` is the
> lightweight path for projects that have no such pipeline.
