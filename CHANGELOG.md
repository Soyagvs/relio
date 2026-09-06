# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0/).

## [1.1.0] - 2026-09-06

### Added

- image: Story-size vertical (1080×1920) + guaranteed fit for many commits

### Changed

- ui: Back to the block wordmark; cat reduced to faint eyes over the O
- ui: Banner is 'Relio' (white) with an orange 'o' and a cat peeking over it
- ui: Draw the banner O as a snake eye with a vertical slit
- ui: Redo the banner O as a filled snake eye (lens slit + catchlight)

## [1.0.0] - 2026-09-06

### Added

- Add 'relio image' — save a shareable PNG of a release
- Add 'relio status'
- Add --no-hash and restyle the release-text header
- Initial Go Release CLI with interactive menu
- Show each commit hash next to its note in the release preview
- image: Add --upload — temp public link + terminal QR code
- image: Drop the RELIO wordmark, ask whether to show commit hashes
- image: Offer the upload+QR after saving when run interactively
- image: Redesign the release card — dark dev identity + git graph + themes
- image: Wrap long changelog lines instead of truncating them
- menu: Add releases browser and help screen
- menu: Head each version in the releases browser with project, version and datetime
- menu: Make the menu one-shot and add a Release text picker
- menu: Press enter in the releases browser to print a version and exit
- menu: Show commit hashes in the releases browser
- menu: Spell out the d-to-delete shortcut in the releases browser
- post: Add the 'social' format
- post: Generate minimal copy-paste release text
- post: Make minimal identical to the releases browser and colour every format
- post: Show the commit range in the minimal text

### Changed

- **Breaking:** Rename Go Release to Relio
- image: Footer reads "generated with relio" instead of the author
- image: Remove the git branch-graph decoration
- ui: Put the commit hash first, in purple, on each note line
- ui: Rebrand to a two-tone orange and purple banner
- ui: Render the RELIO wordmark all orange
