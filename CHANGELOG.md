# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0/).

## [1.1.0] - 2026-09-06

### Added

- Add 'relio status'
- image: Add --upload — temp public link + terminal QR code
- image: Drop the RELIO wordmark, ask whether to show commit hashes
- image: Offer the upload+QR after saving when run interactively
- image: Redesign the release card — dark dev identity + git graph + themes

### Changed

- image: Footer reads "generated with relio" instead of the author

## [1.0.0] - 2026-09-06

### Added

- Add 'relio image' — save a shareable PNG of a release
- Add --no-hash and restyle the release-text header
- post: Add the 'social' format
- post: Make minimal identical to the releases browser and colour every format

### Changed

- **Breaking:** Rename Go Release to Relio
- ui: Render the RELIO wordmark all orange

## [0.4.0] - 2026-09-06

### Added

- menu: Head each version in the releases browser with project, version and datetime
- menu: Make the menu one-shot and add a Release text picker
- menu: Press enter in the releases browser to print a version and exit
- menu: Show commit hashes in the releases browser
- menu: Spell out the d-to-delete shortcut in the releases browser

### Changed

- ui: Put the commit hash first, in purple, on each note line

## [0.3.0] - 2026-09-06

### Added

- Show each commit hash next to its note in the release preview

## [0.2.0] - 2026-09-06

### Added

- post: Show the commit range in the minimal text

## [0.1.0] - 2026-09-06

### Added

- Initial Relio CLI with interactive menu
- menu: Add releases browser and help screen
- post: Generate minimal copy-paste release text

### Changed

- ui: Rebrand to a two-tone orange and purple banner
