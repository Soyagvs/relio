package i18n

// Message IDs, grouped by the surface that uses them. Each constant's value
// is the catalog key. Every catalog in this package MUST define every key
// declared here — enforced by TestCatalogKeyParity and
// TestKeysDeclaredInASTExistInEnglishCatalog in catalog_test.go.
const (
	// Release plan preview (ui.PlanBox / ui.PlanView). This is the only
	// surface localized in this slice; every other surface is converted in
	// later slices.
	PlanCurrentVersion MessageID = "plan.current_version"
	PlanDetectedChange MessageID = "plan.detected_change"
	PlanNextVersion    MessageID = "plan.next_version"
	PlanForced         MessageID = "plan.forced"
	PlanPrerelease     MessageID = "plan.prerelease"
	PlanFinalize       MessageID = "plan.finalize"
	PlanCommitsSince   MessageID = "plan.commits_since" // "%d commits since %s"

	// Settings screen (internal/settings) and its one menu entry. This is a
	// greenfield surface authored i18n-native from birth — not converted from
	// a later slice.
	MenuSettingsLabel MessageID = "menu.settings_label"
	MenuSettingsDesc  MessageID = "menu.settings_desc"

	SettingsTitle                MessageID = "settings.title"
	SettingsLanguageSection      MessageID = "settings.language_section"
	SettingsFooterSection        MessageID = "settings.footer_section"
	SettingsContributorsLabel    MessageID = "settings.contributors_label"
	SettingsContributorsDesc     MessageID = "settings.contributors_desc"
	SettingsCompareLinkLabel     MessageID = "settings.compare_link_label"
	SettingsCompareLinkDesc      MessageID = "settings.compare_link_desc"
	SettingsFooterDisabledReason MessageID = "settings.footer_disabled_reason"
	SettingsHint                 MessageID = "settings.hint"

	// Pick (internal/pick) — the reusable single-select list Bubble Tea
	// component used for sub-choices reached from the main menu. Only the
	// component's own chrome is covered here: item Label/Desc/Value strings
	// are supplied by callers and localized in those callers' own surfaces.
	PickCancelled MessageID = "pick.cancelled"
	PickHint      MessageID = "pick.hint"
)
