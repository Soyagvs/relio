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
