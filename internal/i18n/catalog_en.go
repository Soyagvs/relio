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
}
