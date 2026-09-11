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
}
