# Configuration — `.release.yaml`

Written by `relio init`, read from the repository root. **Configuration only —
no secrets.** A minimal file with just `project:` works; every field below falls
back to the default shown.

```yaml
# .release.yaml — safe to commit. Never put secrets or tokens here.

project: azeink            # free text — shown on cards, posts, and the banner
versioning: semver         # only "semver" is supported
commits: conventional      # only "conventional" is supported
language: ""               # "" = inherit the global preference; "en" / "es" to pin this repo

release:
    # changelog: and tag: are omit-default-true — leave them out to keep both
    # on; set either to false to turn that step off.
    changelog: true                 # write the changelog section on release
    changelog_file: CHANGELOG.md    # which file to write
    tag: true                       # commit the release files and create the git tag
    tag_prefix: v                   # "" for bare 1.6.0 tags instead of v1.6.0
    compare_link: false             # append a GitHub compare link to each changelog section
    contributors: false             # append a "Thanks to …" line built from commit authors

    # Files to rewrite with the new version, in the chore(release) commit.
    # A bare filename uses a built-in rule; a {path, pattern} entry gives an
    # explicit Go regexp with exactly one capture group around the version.
    version_files:
        - package.json
        - Cargo.toml
        - VERSION
        - { path: src/app/__init__.py, pattern: '__version__ = "([^"]+)"' }

    # Shell commands run around a release. `validate` and `before` can abort it; `after` only warns.
    hooks:
        before: make test                   # a string, or a list run in order
        after:
            - ./scripts/notify.sh
            - echo "shipped $RELIO_TAG"

github:
    enabled: false         # reserved — not read yet
    repo: ""               # "owner/name" to publish to; empty = derive from origin
    release: false         # true = every `relio` also publishes the GitHub Release (same as --publish)

content:
    enabled: false         # reserved for the content generator (`relio post`)
```

Unknown `versioning` / `commits` values are rejected with a clear error. `omit
version_files` and `hooks` entirely if you don't use them — they carry no
defaults.
