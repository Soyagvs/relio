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

	// Main menu (internal/menu) chrome: the headline, the digit-jump hint,
	// and every item's label/desc. MenuSettingsLabel/Desc predate this block
	// (added greenfield with the Settings screen itself); the rest were
	// converted from hardcoded English literals in slice 4b.
	MenuHeadline MessageID = "menu.headline" // "%s menu" — %s is the uppercased app name

	MenuHint MessageID = "menu.hint"

	MenuReleaseLabel      MessageID = "menu.release_label"
	MenuReleaseDesc       MessageID = "menu.release_desc"
	MenuStatusLabel       MessageID = "menu.status_label"
	MenuStatusDesc        MessageID = "menu.status_desc"
	MenuCheckLabel        MessageID = "menu.check_label"
	MenuCheckDesc         MessageID = "menu.check_desc"
	MenuViewReleasesLabel MessageID = "menu.view_releases_label"
	MenuViewReleasesDesc  MessageID = "menu.view_releases_desc"
	MenuReleaseTextLabel  MessageID = "menu.release_text_label"
	MenuReleaseTextDesc   MessageID = "menu.release_text_desc"
	MenuReleaseImageLabel MessageID = "menu.release_image_label"
	MenuReleaseImageDesc  MessageID = "menu.release_image_desc"
	MenuAuthLabel         MessageID = "menu.auth_label"
	MenuAuthDesc          MessageID = "menu.auth_desc"
	MenuSetupLabel        MessageID = "menu.setup_label"
	MenuSetupDesc         MessageID = "menu.setup_desc"
	MenuGuideLabel        MessageID = "menu.guide_label"
	MenuGuideDesc         MessageID = "menu.guide_desc"
	MenuHelpLabel         MessageID = "menu.help_label"
	MenuHelpDesc          MessageID = "menu.help_desc"
	MenuExitLabel         MessageID = "menu.exit_label"
	MenuExitDesc          MessageID = "menu.exit_desc"

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

	// Remaining ui.go chrome (PR4c-2): the entry banner's tagline and
	// "created by" credit, versionLabel's "dev build" fallback, and
	// PlanView's hooks/version-files section labels plus the "since the
	// beginning" fallback used when there is no prior tag. ui.Notes,
	// ui.ReleaseText, ui.ReleaseHeader, and ui.ReleaseMeta are the artifact
	// English boundary and are NEVER routed through these or any other
	// i18n.T call — see internal/ui/artifact_invariance_test.go.
	BannerTagline         MessageID = "banner.tagline"
	BannerCreatedBy       MessageID = "banner.created_by"
	BannerDevBuild        MessageID = "banner.dev_build"
	BannerUpdateAvailable MessageID = "banner.update_available" // "▲ v%s available"

	PlanHooksLabel         MessageID = "plan.hooks_label"
	PlanHooksBefore        MessageID = "plan.hooks_before"
	PlanHooksAfter         MessageID = "plan.hooks_after"
	PlanHooksCommandsCount MessageID = "plan.hooks_commands_count" // "%d commands"
	PlanVersionFilesLabel  MessageID = "plan.version_files_label"
	PlanSinceBeginning     MessageID = "plan.since_beginning"

	// Wizard (internal/wizard) — the interactive release-confirmation
	// stepper: choice labels, header, bump-from note, footer hint, and the
	// two post-quit trace lines (PR5a).
	WizardChoiceConfirm MessageID = "wizard.choice_confirm" // "Create %s"
	WizardChoicePatch   MessageID = "wizard.choice_patch"
	WizardChoiceMinor   MessageID = "wizard.choice_minor"
	WizardChoiceMajor   MessageID = "wizard.choice_major"
	WizardChoiceCancel  MessageID = "wizard.choice_cancel"
	WizardHeader        MessageID = "wizard.header"    // "Release %s"
	WizardBumpFrom      MessageID = "wizard.bump_from" // "(%s from %s)"
	WizardConfirmed     MessageID = "wizard.confirmed" // "→ confirmed %s"
	WizardCancelled     MessageID = "wizard.cancelled"
	WizardHint          MessageID = "wizard.hint"

	// Releases (internal/releases) — the interactive release browser: title,
	// empty-state message, delete-confirmation prompt, delete outcome status
	// lines, the notesFor fallback placeholder shown when no notes exist at
	// all, and footer hints (PR5b). notesFor's ui.Notes/ui.ReleaseText/
	// ui.ReleaseHeader/ui.ReleaseMeta path is NEVER routed through these or
	// any other i18n.T call — see internal/ui/artifact_invariance_test.go.
	ReleasesTitle           MessageID = "releases.title"
	ReleasesEmpty           MessageID = "releases.empty"
	ReleasesEmptyHint       MessageID = "releases.empty_hint"
	ReleasesNoNotes         MessageID = "releases.no_notes"
	ReleasesNonePlaceholder MessageID = "releases.none_placeholder"

	ReleasesDeleteConfirm          MessageID = "releases.delete_confirm" // "Delete %s? This removes the git tag and its changelog section.  [y/N]"
	ReleasesDeleteCancelled        MessageID = "releases.delete_cancelled"
	ReleasesDeleted                MessageID = "releases.deleted" // "Deleted %s (%s)."
	ReleasesRemovedTag             MessageID = "releases.removed_tag"
	ReleasesRemovedTagAndChangelog MessageID = "releases.removed_tag_and_changelog"
	ReleasesChangelogWriteError    MessageID = "releases.changelog_write_error" // "tag deleted, but changelog: %v"

	ReleasesHintPrintNotesExit MessageID = "releases.hint_print_notes_exit"
	ReleasesHintDeleteRelease  MessageID = "releases.hint_delete_release"
	ReleasesHintMove           MessageID = "releases.hint_move"
	ReleasesHintShowExit       MessageID = "releases.hint_show_exit"
	ReleasesHintDelete         MessageID = "releases.hint_delete"
	ReleasesHintBack           MessageID = "releases.hint_back"
)
