# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0/).

## [Unreleased]

### Added

- `go-release` CLI scaffold built with Cobra and Lipgloss.
- Interactive main menu (Bubble Tea): Create a release / GitHub auth / Exit,
  with a figlet-style "Go Release" banner and a `created by SOYAGVS` credit.
- Conventional Commits parser.
- SemVer calculator with `--patch` / `--minor` / `--major` overrides.
- `CHANGELOG.md` generation in the Keep a Changelog format.
- Git tag creation with a preview before anything is written.
- Interactive confirmation wizard (Bubble Tea) for the release step.
- `go-release init` to scaffold `.release.yaml`.
- `go-release post` experimental announcement generator (technical / casual / changelog).
- `go-release auth` command surface as a v0.2.0 placeholder.
