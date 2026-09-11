package i18n

// en is the reference catalog: every other catalog must define every key
// defined here (see catalog_test.go).
var en = map[MessageID]string{
	PlanCurrentVersion: "Current version",
	PlanDetectedChange: "Detected change",
	PlanNextVersion:    "Next version",
	PlanForced:         " (forced)",
	PlanPrerelease:     " (pre-release)",
	PlanFinalize:       " (finalize)",
	PlanCommitsSince:   "%d commits since %s",

	MenuHeadline: "%s menu",
	MenuHint:     "↑/↓ move · 1–9 jump · ? help · g guide · enter select · q quit",

	MenuReleaseLabel:      "Release",
	MenuReleaseDesc:       "Create a release — final or rc, and optionally push + publish",
	MenuStatusLabel:       "Status",
	MenuStatusDesc:        "What's unreleased and the version it suggests",
	MenuCheckLabel:        "Check",
	MenuCheckDesc:         "Which commits since the last tag are Conventional Commits",
	MenuViewReleasesLabel: "Releases",
	MenuViewReleasesDesc:  "Browse versions, read notes, delete one",
	MenuReleaseTextLabel:  "Announcement",
	MenuReleaseTextDesc:   "Copy-paste release text — pick a format",
	MenuReleaseImageLabel: "Release image",
	MenuReleaseImageDesc:  "Save or share a PNG release card",
	MenuAuthLabel:         "Auth",
	MenuAuthDesc:          "GitHub connection — status and how to link",
	MenuSetupLabel:        "Setup",
	MenuSetupDesc:         "Create or inspect .release.yaml",
	MenuGuideLabel:        "Guide",
	MenuGuideDesc:         "Step-by-step walkthrough of the whole flow",
	MenuHelpLabel:         "Help",
	MenuHelpDesc:          "Every command and flag",
	MenuExitLabel:         "Exit",
	MenuExitDesc:          "Leave Relio",

	MenuSettingsLabel: "Settings",
	MenuSettingsDesc:  "Language and release-footer preferences",

	SettingsTitle:                "Settings",
	SettingsLanguageSection:      "Language",
	SettingsFooterSection:        "Changelog footer",
	SettingsContributorsLabel:    "Contributors line",
	SettingsContributorsDesc:     "Add a contributors line to the changelog footer",
	SettingsCompareLinkLabel:     "Compare link",
	SettingsCompareLinkDesc:      "Add a GitHub compare link to the changelog footer",
	SettingsFooterDisabledReason: "Open a project with .release.yaml to edit these",
	SettingsHint:                 "↑/↓ move · space toggle · q back",

	PickCancelled: "→ cancelled",
	PickHint:      "↑/↓ move · enter select · q cancel",

	BannerTagline:         "turn commits into releases",
	BannerCreatedBy:       "created by",
	BannerDevBuild:        "dev build",
	BannerUpdateAvailable: "▲ v%s available",

	PlanHooksLabel:         "hooks",
	PlanHooksBefore:        "before: ",
	PlanHooksAfter:         "after: ",
	PlanHooksCommandsCount: "%d commands",
	PlanVersionFilesLabel:  "Version files",
	PlanSinceBeginning:     "the beginning",

	WizardChoiceConfirm: "Create %s",
	WizardChoicePatch:   "Change to patch",
	WizardChoiceMinor:   "Change to minor",
	WizardChoiceMajor:   "Change to major",
	WizardChoiceCancel:  "Cancel",
	WizardHeader:        "Release %s",
	WizardBumpFrom:      "(%s from %s)",
	WizardConfirmed:     "→ confirmed %s",
	WizardCancelled:     "→ cancelled",
	WizardHint:          "↑/↓ move · enter select · y confirm · q cancel",

	ReleasesTitle:           "Releases",
	ReleasesEmpty:           "No releases yet. Create one from the menu.",
	ReleasesEmptyHint:       "q back",
	ReleasesNoNotes:         "(no notes for this version)",
	ReleasesNonePlaceholder: "(none)",

	ReleasesDeleteConfirm:          "Delete %s? This removes the git tag and its changelog section.  [y/N]",
	ReleasesDeleteCancelled:        "Delete cancelled.",
	ReleasesDeleted:                "Deleted %s (%s).",
	ReleasesRemovedTag:             "tag",
	ReleasesRemovedTagAndChangelog: "tag + changelog section",
	ReleasesChangelogWriteError:    "tag deleted, but changelog: %v",

	ReleasesHintPrintNotesExit: "print notes & exit",
	ReleasesHintDeleteRelease:  "delete release",
	ReleasesHintMove:           "move",
	ReleasesHintShowExit:       "show & exit",
	ReleasesHintDelete:         "delete",
	ReleasesHintBack:           "back",

	GuideStep1Title:      "What Relio does",
	GuideStep1Body:       "The flow is:  code → commit → push → relio → version + CHANGELOG + tag. Nothing is written until you confirm the preview, and Relio never pushes on its own unless you ask it to.",
	GuideStep1NoRepoHint: "Run this inside a git repository to follow the steps below.",

	GuideStep2Title: "Set up `.release.yaml`",
	GuideStep2ConfiguredBody: "Already set up (project: %s), so you can skip `relio init`. This is " +
		"Relio's own config at the repo root — not your package.json / pyproject.toml. " +
		"Configuration only, never secrets: changelog file, tag prefix, and optionally " +
		"version_files and hooks.",
	GuideStep2UnconfiguredBody: "Relio needs its own file, `.release.yaml`, at the repo root — separate from any " +
		"version file your language already has (package.json, pyproject.toml, …), and always " +
		"read from the project root no matter where you run relio from. Configuration only, " +
		"never secrets: changelog file, tag prefix, and optionally version_files (list those " +
		"language files here to keep them in sync) and hooks. Create it with `relio init`, or " +
		"hand-write a minimal one — just `project: <name>` works.",

	GuideStep3Title: "Write Conventional Commits",
	GuideStep3Body: "`feat:` bumps the minor; `fix:` / `perf:` / `refactor:` bump the patch; " +
		"`feat!:` or a `BREAKING CHANGE:` footer bumps the major. Commits without a type are ignored for versioning.",
	GuideStep3CheckHint: "Run `relio check` to see which of your commits qualify.",

	GuideStep4Title: "See what's pending",
	GuideStep4Body:  "`relio status` lists the unreleased commits and the version they suggest.",

	GuideStep5Title: "Create the release",
	GuideStep5Body: "Run `relio` with no arguments: you get a preview, then a small wizard " +
		"(Create / change the bump / cancel). On confirm it writes the CHANGELOG.md section, " +
		"commits it as `chore(release): vX.Y.Z`, and creates the annotated tag — nothing before the confirm.\n" +
		"`relio --rc` cuts a release candidate you can iterate on; running `relio` again on an rc finalizes it.",

	GuideStep6Title: "Get it out",
	GuideStep6Body: "Push with `git push --follow-tags`. Or `relio --publish` to push and create the GitHub Release " +
		"with the changelog notes as its body — that needs a GitHub token (GITHUB_TOKEN / GH_TOKEN / `gh auth login`).",
	GuideStep6NoTokenHint: "No GitHub token is set yet.",

	GuideStep7Title: "Optional extras",
	GuideStep7Body: "`relio post` prints announcement text for socials. `relio image` renders a PNG release card. " +
		"`.release.yaml` `version_files:` writes the new version into package.json / pyproject.toml / …. " +
		"`.release.yaml` `release.hooks.before` / `.after` run shell commands around the release.",

	GuideStep8Title: "You're set",
	GuideStep8Body: "Happy path:  %s.\n" +
		"See `relio help` for every command and flag, and the README for the full `.release.yaml` reference.",

	GuideActionRunInit:   "run relio init now",
	GuideActionRunCheck:  "run relio check now",
	GuideActionRunStatus: "run relio status now",

	GuideHappyPath:   "relio init → write feat:/fix: commits → relio status → relio → git push --follow-tags",
	GuidePlainHeader: "Relio — guide",
	GuidePlainFooter: "The happy path:  %s",

	GuideStepCounter:      "Step %d of %d",
	GuideFooterWithAction: "[y] %s · enter skip · ← back · q quit",
	GuideFooterNoAction:   "enter continue · ← back · q quit",

	RootShort: "Turn finished code into a published release",
	RootLong: "  Read the repo's git activity and turn it into a version, changelog,\n" +
		"  and tag — in one command, with a preview before anything is written.\n\n" +
		"  Run `relio` on its own for the interactive menu. Use the\n" +
		"  subcommands below for setup, extras, and scripting.",

	FlagDirUsage:            "run as if relio was started in `path`",
	FlagNoHashUsage:         "hide commit hashes in release notes",
	FlagPatchUsage:          "force a PATCH bump",
	FlagMinorUsage:          "force a MINOR bump",
	FlagMajorUsage:          "force a MAJOR bump",
	FlagYesUsage:            "skip the interactive menu and confirmation",
	FlagNoChangelogUsage:    "do not touch the changelog file",
	FlagNoTagUsage:          "do not create the git tag",
	FlagNoVersionFilesUsage: "do not update the files listed in version_files",
	FlagPublishUsage:        "push and create the GitHub Release after tagging",
	FlagRCUsage:             "cut a release candidate (vX.Y.Z-rc.N) instead of the final version",
	FlagNoHooksUsage:        "skip the before/after hooks in .release.yaml for this run",
	FlagEditUsage:           "open the generated release notes in your editor before writing",

	VersionInfoLine:        "%s %s (commit %s, built %s)\n",
	VersionShort:           "Print the Relio version",
	VersionUpdateAvailable: "▲ v%s available — brew upgrade relio",

	HelpSectionCommands:     "Commands",
	HelpSectionReleaseFlags: "Release flags",
	HelpSectionPostFlags:    "post flags",
	HelpSectionImageFlags:   "image flags",
	HelpSectionMenu:         "Menu",
	HelpFooter:              "Conventional Commits drive the version: fix→patch, feat→minor, feat!/BREAKING→major.",

	HelpCmdRelioDesc:  "Create a release: version + changelog + tag from commits since the last tag",
	HelpCmdStatusDesc: "Show what's unreleased since the last tag and the version it suggests",
	HelpCmdCheckDesc:  "List which commits since the last tag are Conventional Commits (--strict)",
	HelpCmdGuideDesc:  "Walk through the whole release flow step by step",
	HelpCmdStatsDesc:  "Relio's public GitHub download stats (read-only, no telemetry)",
	HelpCmdInitDesc:   "Create .release.yaml in the current repo (configuration only, never secrets)",
	HelpCmdPostDesc:   "Print copy-paste release text for social posts (text on stdout only)",
	HelpCmdImageDesc:  "Make a release card image — save it, upload it for a link, or both (--shape, --theme, --hash, --upload, --link-only)",
	HelpCmdAuthDesc:   "Inspect the GitHub token relio will use (`relio auth status`)",

	HelpFlagBumpDesc:        "Force the version bump instead of inferring it from the commits",
	HelpFlagYesDesc:         "Skip the menu and the confirmation (required in CI or a non-interactive shell)",
	HelpFlagNoChangelogDesc: "Do not modify the changelog file",
	HelpFlagNoTagDesc:       "Do not create the git tag",
	HelpFlagRCDesc:          "Cut a release candidate (vX.Y.Z-rc.N); run `relio` on an rc to finalize it",
	HelpFlagPublishDesc:     "After tagging, push the branch and tag to origin and create the GitHub Release",
	HelpFlagNoHashDesc:      "Hide the commit hash on each release-note line",
	HelpFlagDirDesc:         "Run as if Relio was started in <path>",

	HelpPostMinimalDesc:   "Same output as the releases browser: project, version, date, commits, grouped notes (default)",
	HelpPostSocialDesc:    "Shortest: \"Project -- Release\", version · date · time, then \"type  description\" lines",
	HelpPostTechnicalDesc: "Terse bullet list, for a changelog or a dev channel",
	HelpPostCasualDesc:    "Loose tone: \"proj v1.4.0 is out. → …\"",
	HelpPostChangelogDesc: "The exact section that goes into CHANGELOG.md",

	HelpImageShapeDesc:    "horizontal (1200×630) | vertical (1080×1920) | square (1080×1080)",
	HelpImageThemeDesc:    "orange (default) | green | purple accent",
	HelpImageHashDesc:     "show the commit hash on each line",
	HelpImageUploadDesc:   "also upload to a temp host (litterbox 72h) and print a link + QR",
	HelpImageLinkOnlyDesc: "upload for a link + QR without writing a local file",

	HelpMenuReleaseDesc:      "Pick final vs rc and whether to publish, then run `relio`",
	HelpMenuStatusDesc:       "What's unreleased and the suggested version (same as `relio status`)",
	HelpMenuCheckDesc:        "Which commits since the last tag are Conventional Commits (same as `relio check`)",
	HelpMenuReleasesDesc:     "List versions, read a version's notes, or delete one (git tag + changelog section)",
	HelpMenuAnnouncementDesc: "Pick a post format and print copy-paste text (same as `relio post`)",
	HelpMenuReleaseImageDesc: "Pick a release + shape, then save the card, upload it for a link, or both (same as `relio image`)",
	HelpMenuAuthDesc:         "GitHub connection — status and how to link (see `relio auth status`)",
	HelpMenuSetupDesc:        "Create or inspect .release.yaml (same as `relio init`)",
	HelpMenuGuideDesc:        "Step-by-step walkthrough (same as `relio guide`)",
	HelpMenuHelpDesc:         "This screen",

	StatusShort: "Show what's unreleased and the version it suggests",

	StatusLabelCurrent:    "Current",
	StatusLabelUnreleased: "Unreleased",
	StatusLabelSuggested:  "Suggested",
	StatusCommitsCount:    "%d commits",
	StatusNoSuggestion:    "—",

	StatusNothingToRelease:  "Nothing to release.",
	StatusPrereleaseHint:    "on a pre-release — `relio` finalizes %s, `relio --rc` cuts the next rc",
	StatusUncommittedChange: "uncommitted changes in the working tree",
	StatusReadyToRelease:    "Ready to release.",

	CheckShort:           "Check the commits since the last tag before releasing",
	CheckFlagStrictUsage: "exit non-zero when any commit is not a Conventional Commit",

	CheckBaseLastTag:         "the last tag",
	CheckNothingToCheck:      "Nothing to check — no commits since %s.",
	CheckCommitsNoTagYet:     "%d commits (no tag yet)",
	CheckCommitsSinceTag:     "%d commits since %s",
	CheckConventionalCount:   "%d conventional",
	CheckNonConventionalHead: "%d not conventional:",
	CheckDetectedBump:        "Detected bump: %s  →  %s",
	CheckStrictError:         "%d commit(s) are not Conventional Commits (--strict)",

	InitShort:            "Create a .release.yaml in the current repository",
	InitLong:             "Write a .release.yaml with sensible defaults. The file holds configuration only — never secrets.",
	InitFlagProjectUsage: "project name (defaults to the repo/remote name)",

	InitNotAGitRepo:   "not a git repository — run `relio init` inside a repo",
	InitAlreadyExists: "%s already exists at %s",
	InitConfigCreated: "%s created",
	InitNextStepsHint: "  Review it, commit it, then run `relio`.",

	PostShort: "Generate copy-paste release text for social posts",
	PostLong: "Build a short, plain-text announcement from commits since the last tag.\n" +
		"Only the text goes to stdout, so `relio post | pbcopy` works cleanly.\n" +
		"Experimental preview of the v0.3.0 content generator — nothing is published.",

	PostNoCommits: "No commits since the last tag — nothing to announce.",
	PostCopyHint:  "# release text — copy from here:",

	PostFormatMinimalLabel:   "Minimal",
	PostFormatMinimalDesc:    "Same as the releases browser: project, version, date, commits, grouped notes with hashes",
	PostFormatSocialLabel:    "Social",
	PostFormatTechnicalLabel: "Technical",
	PostFormatCasualLabel:    "Casual",
	PostFormatChangelogLabel: "Changelog",

	PostUnknownFormat: "unknown format %q (minimal|social|technical|casual|changelog)",

	PostSocialHeader:            "%s -- Release",
	PostSocialNoNotableChanges:  "(no notable changes)",
	PostTechnicalCommitsSummary: "%d commits · %s",
	PostCasualIsOut:             "is out.",

	PublishNoTokenSkip: "Skipping GitHub publish: no token found.",
	PublishNoTokenHint: "  set GITHUB_TOKEN to a PAT with `repo` scope, or run `gh auth login`, then:",

	PublishNoOriginRemote: "cannot publish: no `origin` remote (the tag %s is created locally): %w",
	PublishUnknownRepo:    "cannot tell which GitHub repo to publish to (the tag %s is created locally — set `github.repo` in %s): %w",

	PublishConfirmPrompt:    "  Push %s and %s to origin and publish the GitHub Release? [y/N] ",
	PublishPushBranchFailed: "pushing %s to origin failed (the tag %s is intact locally — retry once the remote is reachable): %w",
	PublishPushTagFailed:    "pushing tag %s to origin failed (the tag is intact locally — retry with `git push origin %s`): %w",
	PublishPushedToOrigin:   "pushed to origin",
	PublishReleaseExists:    "GitHub Release %s already exists — skipping.",
	PublishCreateFailed:     "pushed to origin, but creating the GitHub Release failed (the tag %s is on origin — create the Release from the web UI or re-run): %w",
	PublishReleasePublished: "GitHub Release %s published",

	UndoShort:          "Reverse the most recent local release (before it is pushed)",
	UndoFlagYesUsage:   "skip the confirmation prompt",
	UndoFlagForceUsage: "undo even with a dirty working tree (git reset --hard discards uncommitted changes)",

	UndoNoTags:        "No tags yet — nothing to undo.",
	UndoTagNotAtHead:  "%s does not point at HEAD — the last release is not the current commit, nothing to undo safely",
	UndoAlreadyPushed: "%s is already on a remote — undo would rewrite shared history.\n  remove it on the remote yourself:  git push origin :%s\n  and delete the GitHub Release if you created one",
	UndoDirtyTree:     "the working tree has uncommitted changes — commit or stash them first, or re-run with --force",

	UndoHeader:           "Undo %s",
	UndoStepDeleteTag:    "  · delete the local tag %s",
	UndoStepRemoveCommit: "  · remove the `chore(release): %s` commit (git reset --hard HEAD~1)",
	UndoStepFilesRevert:  "    %s and any version files return to their previous state",
	UndoProceedPrompt:    "  Proceed? [y/N] ",
	UndoCancelled:        "Cancelled. Nothing changed.",

	UndoDoneDeletedTag:    "deleted tag %s",
	UndoDoneRemovedCommit: "removed the release commit",
	UndoResetFailed:       "tag %s deleted, but removing the release commit failed (finish with `git reset --hard HEAD~1`): %w",
}
