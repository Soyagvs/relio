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

	// Guide (internal/guide) — `relio guide`'s eight-step walkthrough, both
	// the interactive Bubble Tea stepper and the plain-text fallback: every
	// step's title/body, the two conditional hints, the three "run it now"
	// action labels, the shared happy-path phrase, the plain-text header and
	// footer, and the interactive stepper's own chrome (step counter, the two
	// footer variants) (PR5c).
	GuideStep1Title            MessageID = "guide.step1_title"
	GuideStep1Body             MessageID = "guide.step1_body"
	GuideStep1NoRepoHint       MessageID = "guide.step1_no_repo_hint"
	GuideStep2Title            MessageID = "guide.step2_title"
	GuideStep2ConfiguredBody   MessageID = "guide.step2_configured_body" // "Already set up (project: %s), ..."
	GuideStep2UnconfiguredBody MessageID = "guide.step2_unconfigured_body"
	GuideStep3Title            MessageID = "guide.step3_title"
	GuideStep3Body             MessageID = "guide.step3_body"
	GuideStep3CheckHint        MessageID = "guide.step3_check_hint"
	GuideStep4Title            MessageID = "guide.step4_title"
	GuideStep4Body             MessageID = "guide.step4_body"
	GuideStep5Title            MessageID = "guide.step5_title"
	GuideStep5Body             MessageID = "guide.step5_body"
	GuideStep6Title            MessageID = "guide.step6_title"
	GuideStep6Body             MessageID = "guide.step6_body"
	GuideStep6NoTokenHint      MessageID = "guide.step6_no_token_hint"
	GuideStep7Title            MessageID = "guide.step7_title"
	GuideStep7Body             MessageID = "guide.step7_body"
	GuideStep8Title            MessageID = "guide.step8_title"
	GuideStep8Body             MessageID = "guide.step8_body" // "Happy path:  %s.\nSee ..."

	GuideActionRunInit   MessageID = "guide.action_run_init"
	GuideActionRunCheck  MessageID = "guide.action_run_check"
	GuideActionRunStatus MessageID = "guide.action_run_status"

	GuideHappyPath   MessageID = "guide.happy_path"
	GuidePlainHeader MessageID = "guide.plain_header"
	GuidePlainFooter MessageID = "guide.plain_footer" // "The happy path:  %s"

	GuideStepCounter      MessageID = "guide.step_counter"       // "Step %d of %d"
	GuideFooterWithAction MessageID = "guide.footer_with_action" // "[y] %s · enter skip · ← back · q quit"
	GuideFooterNoAction   MessageID = "guide.footer_no_action"   // "enter continue · ← back · q quit"

	// Root command (cmd/root.go): Short/Long text and every persistent/local
	// flag's usage string. NewRootCmd() must be called AFTER the active
	// language is resolved — Execute() prescans os.Args for -C/--dir and
	// resolves the language before building the tree, because cobra bakes
	// these strings into the *cobra.Command struct at construction time
	// (PR6a).
	RootShort MessageID = "root.short"
	RootLong  MessageID = "root.long"

	FlagDirUsage            MessageID = "flag.dir_usage"
	FlagNoHashUsage         MessageID = "flag.no_hash_usage"
	FlagPatchUsage          MessageID = "flag.patch_usage"
	FlagMinorUsage          MessageID = "flag.minor_usage"
	FlagMajorUsage          MessageID = "flag.major_usage"
	FlagYesUsage            MessageID = "flag.yes_usage"
	FlagNoChangelogUsage    MessageID = "flag.no_changelog_usage"
	FlagNoTagUsage          MessageID = "flag.no_tag_usage"
	FlagNoVersionFilesUsage MessageID = "flag.no_version_files_usage"
	FlagPublishUsage        MessageID = "flag.publish_usage"
	FlagRCUsage             MessageID = "flag.rc_usage"
	FlagNoHooksUsage        MessageID = "flag.no_hooks_usage"
	FlagEditUsage           MessageID = "flag.edit_usage"

	// Version output shared by cmd/root.go's --version template and
	// cmd/version.go's `relio version` command, plus that command's own
	// Short text and update-available nudge (PR6a).
	VersionInfoLine        MessageID = "version.info_line" // "%s %s (commit %s, built %s)\n"
	VersionShort           MessageID = "version.short"
	VersionUpdateAvailable MessageID = "version.update_available" // "▲ v%s available — brew upgrade relio"

	// Help reference screen (cmd/help.go): section headers, the footer line,
	// and every row's description. Row *names* (command syntax, flag
	// syntax) stay literal English on purpose — they are code the user
	// types verbatim, not prose. Menu-row names reuse the existing
	// MenuXLabel keys from internal/menu's own conversion instead of
	// duplicating them (PR6a).
	HelpSectionCommands     MessageID = "help.section_commands"
	HelpSectionReleaseFlags MessageID = "help.section_release_flags"
	HelpSectionPostFlags    MessageID = "help.section_post_flags"
	HelpSectionImageFlags   MessageID = "help.section_image_flags"
	HelpSectionMenu         MessageID = "help.section_menu"
	HelpFooter              MessageID = "help.footer"

	HelpCmdRelioDesc  MessageID = "help.cmd_relio_desc"
	HelpCmdStatusDesc MessageID = "help.cmd_status_desc"
	HelpCmdCheckDesc  MessageID = "help.cmd_check_desc"
	HelpCmdGuideDesc  MessageID = "help.cmd_guide_desc"
	HelpCmdStatsDesc  MessageID = "help.cmd_stats_desc"
	HelpCmdInitDesc   MessageID = "help.cmd_init_desc"
	HelpCmdPostDesc   MessageID = "help.cmd_post_desc"
	HelpCmdImageDesc  MessageID = "help.cmd_image_desc"
	HelpCmdAuthDesc   MessageID = "help.cmd_auth_desc"

	HelpFlagBumpDesc        MessageID = "help.flag_bump_desc"
	HelpFlagYesDesc         MessageID = "help.flag_yes_desc"
	HelpFlagNoChangelogDesc MessageID = "help.flag_no_changelog_desc"
	HelpFlagNoTagDesc       MessageID = "help.flag_no_tag_desc"
	HelpFlagRCDesc          MessageID = "help.flag_rc_desc"
	HelpFlagPublishDesc     MessageID = "help.flag_publish_desc"
	HelpFlagNoHashDesc      MessageID = "help.flag_no_hash_desc"
	HelpFlagDirDesc         MessageID = "help.flag_dir_desc"

	HelpPostMinimalDesc   MessageID = "help.post_minimal_desc"
	HelpPostSocialDesc    MessageID = "help.post_social_desc"
	HelpPostTechnicalDesc MessageID = "help.post_technical_desc"
	HelpPostCasualDesc    MessageID = "help.post_casual_desc"
	HelpPostChangelogDesc MessageID = "help.post_changelog_desc"

	HelpImageShapeDesc    MessageID = "help.image_shape_desc"
	HelpImageThemeDesc    MessageID = "help.image_theme_desc"
	HelpImageHashDesc     MessageID = "help.image_hash_desc"
	HelpImageUploadDesc   MessageID = "help.image_upload_desc"
	HelpImageLinkOnlyDesc MessageID = "help.image_link_only_desc"

	HelpMenuReleaseDesc      MessageID = "help.menu_release_desc"
	HelpMenuStatusDesc       MessageID = "help.menu_status_desc"
	HelpMenuCheckDesc        MessageID = "help.menu_check_desc"
	HelpMenuReleasesDesc     MessageID = "help.menu_releases_desc"
	HelpMenuAnnouncementDesc MessageID = "help.menu_announcement_desc"
	HelpMenuReleaseImageDesc MessageID = "help.menu_release_image_desc"
	HelpMenuAuthDesc         MessageID = "help.menu_auth_desc"
	HelpMenuSetupDesc        MessageID = "help.menu_setup_desc"
	HelpMenuGuideDesc        MessageID = "help.menu_guide_desc"
	HelpMenuHelpDesc         MessageID = "help.menu_help_desc"

	// Status command (cmd/status.go): Short text, the three row labels,
	// the commit-count template, the "no suggestion" placeholder, the
	// pre-release finalize hint, and the terminal status lines (PR6b).
	StatusShort MessageID = "status.short"

	StatusLabelCurrent    MessageID = "status.label_current"
	StatusLabelUnreleased MessageID = "status.label_unreleased"
	StatusLabelSuggested  MessageID = "status.label_suggested"
	StatusCommitsCount    MessageID = "status.commits_count" // "%d commits"
	StatusNoSuggestion    MessageID = "status.no_suggestion" // "—"

	StatusNothingToRelease  MessageID = "status.nothing_to_release"
	StatusPrereleaseHint    MessageID = "status.prerelease_hint" // "on a pre-release — `relio` finalizes %s, `relio --rc` cuts the next rc"
	StatusUncommittedChange MessageID = "status.uncommitted_change"
	StatusReadyToRelease    MessageID = "status.ready_to_release"

	// Check command (cmd/check.go): Short text, the --strict flag usage,
	// the "the last tag" fallback, the header lines, the conventional /
	// non-conventional counts, the detected-bump line, and the --strict
	// error (PR6b).
	CheckShort           MessageID = "check.short"
	CheckFlagStrictUsage MessageID = "check.flag_strict_usage"

	CheckBaseLastTag         MessageID = "check.base_last_tag"
	CheckNothingToCheck      MessageID = "check.nothing_to_check"      // "Nothing to check — no commits since %s."
	CheckCommitsNoTagYet     MessageID = "check.commits_no_tag_yet"    // "%d commits (no tag yet)"
	CheckCommitsSinceTag     MessageID = "check.commits_since_tag"     // "%d commits since %s"
	CheckConventionalCount   MessageID = "check.conventional_count"    // "%d conventional"
	CheckNonConventionalHead MessageID = "check.non_conventional_head" // "%d not conventional:"
	CheckDetectedBump        MessageID = "check.detected_bump"         // "Detected bump: %s  →  %s"
	CheckStrictError         MessageID = "check.strict_error"          // "%d commit(s) are not Conventional Commits (--strict)"

	// Init command (cmd/init.go): Short/Long text, the --project flag
	// usage, the not-a-git-repo and already-exists errors, the created
	// confirmation, and the next-steps hint (PR6b).
	InitShort            MessageID = "init.short"
	InitLong             MessageID = "init.long"
	InitFlagProjectUsage MessageID = "init.flag_project_usage"

	InitNotAGitRepo   MessageID = "init.not_a_git_repo"
	InitAlreadyExists MessageID = "init.already_exists" // "%s already exists at %s"
	InitConfigCreated MessageID = "init.config_created" // "%s created"
	InitNextStepsHint MessageID = "init.next_steps_hint"

	// Post command (cmd/post.go): Short/Long text, the "nothing to
	// announce" info line, the copy-from-here hint, the format picker's
	// item labels (postFormats was rebuilt as postFormatItems(), a
	// call-time function, for the same reason help.go's tables were —
	// a package-level var would freeze at "en" before Execute() ever
	// resolves the language), the unknown-format error, and the
	// per-format literal chrome (social header/no-notable-changes,
	// technical commit summary, casual "is out."). The --format flag's
	// own usage string lists literal values the user types verbatim
	// (minimal|social|technical|casual|changelog) and stays untranslated
	// on purpose, matching cmd/help.go's row-name convention (PR6c).
	// Four of the five format-item descriptions are byte-identical to
	// cmd/help.go's HelpPost*Desc keys (PR6a) and reuse them instead of
	// duplicating; only "Minimal" differs in wording between the two
	// surfaces and gets its own key.
	PostShort MessageID = "post.short"
	PostLong  MessageID = "post.long"

	PostNoCommits MessageID = "post.no_commits"
	PostCopyHint  MessageID = "post.copy_hint"

	PostFormatMinimalLabel   MessageID = "post.format_minimal_label"
	PostFormatMinimalDesc    MessageID = "post.format_minimal_desc"
	PostFormatSocialLabel    MessageID = "post.format_social_label"
	PostFormatTechnicalLabel MessageID = "post.format_technical_label"
	PostFormatCasualLabel    MessageID = "post.format_casual_label"
	PostFormatChangelogLabel MessageID = "post.format_changelog_label"

	PostUnknownFormat MessageID = "post.unknown_format" // "unknown format %q (minimal|social|technical|casual|changelog)"

	PostSocialHeader            MessageID = "post.social_header" // "%s -- Release"
	PostSocialNoNotableChanges  MessageID = "post.social_no_notable_changes"
	PostTechnicalCommitsSummary MessageID = "post.technical_commits_summary" // "%d commits · %s"
	PostCasualIsOut             MessageID = "post.casual_is_out"

	// Publish command (cmd/publish.go): publishGitHubRelease's every
	// printed line and wrapped error — the no-token skip notice and hint,
	// the unresolved-repo errors, the push confirmation prompt, the push
	// failure errors, the success lines, the already-exists notice, and
	// the create-failure error. The literal `git push && git push origin
	// <tag>` hint line stays untranslated — it is a command the user
	// types verbatim, not prose (PR6c).
	PublishNoTokenSkip MessageID = "publish.no_token_skip"
	PublishNoTokenHint MessageID = "publish.no_token_hint"

	PublishNoOriginRemote MessageID = "publish.no_origin_remote" // "cannot publish: no `origin` remote (the tag %s is created locally): %w"
	PublishUnknownRepo    MessageID = "publish.unknown_repo"     // "cannot tell which GitHub repo to publish to (the tag %s is created locally — set `github.repo` in %s): %w"

	PublishConfirmPrompt    MessageID = "publish.confirm_prompt"     // "  Push %s and %s to origin and publish the GitHub Release? [y/N] "
	PublishPushBranchFailed MessageID = "publish.push_branch_failed" // "pushing %s to origin failed (the tag %s is intact locally — retry once the remote is reachable): %w"
	PublishPushTagFailed    MessageID = "publish.push_tag_failed"    // "pushing tag %s to origin failed (the tag is intact locally — retry with `git push origin %s`): %w"
	PublishPushedToOrigin   MessageID = "publish.pushed_to_origin"
	PublishReleaseExists    MessageID = "publish.release_exists"    // "GitHub Release %s already exists — skipping."
	PublishCreateFailed     MessageID = "publish.create_failed"     // "pushed to origin, but creating the GitHub Release failed (the tag %s is on origin — create the Release from the web UI or re-run): %w"
	PublishReleasePublished MessageID = "publish.release_published" // "GitHub Release %s published"

	// Undo command (cmd/undo.go): Short text, the --yes/--force flag
	// usage strings, the no-tags info line, the tag-not-at-head and
	// already-pushed errors, the dirty-tree error, the undo-plan header
	// and its two step lines, the confirmation prompt, the cancelled
	// notice, the done-list entries, and the reset-failure error (PR6c).
	UndoShort          MessageID = "undo.short"
	UndoFlagYesUsage   MessageID = "undo.flag_yes_usage"
	UndoFlagForceUsage MessageID = "undo.flag_force_usage"

	UndoNoTags        MessageID = "undo.no_tags"
	UndoTagNotAtHead  MessageID = "undo.tag_not_at_head" // "%s does not point at HEAD — the last release is not the current commit, nothing to undo safely"
	UndoAlreadyPushed MessageID = "undo.already_pushed"  // "%s is already on a remote — undo would rewrite shared history.\n  remove it on the remote yourself:  git push origin :%s\n  and delete the GitHub Release if you created one"
	UndoDirtyTree     MessageID = "undo.dirty_tree"

	UndoHeader           MessageID = "undo.header"             // "Undo %s"
	UndoStepDeleteTag    MessageID = "undo.step_delete_tag"    // "  · delete the local tag %s"
	UndoStepRemoveCommit MessageID = "undo.step_remove_commit" // "  · remove the `chore(release): %s` commit (git reset --hard HEAD~1)"
	UndoStepFilesRevert  MessageID = "undo.step_files_revert"  // "    %s and any version files return to their previous state"
	UndoProceedPrompt    MessageID = "undo.proceed_prompt"
	UndoCancelled        MessageID = "undo.cancelled"

	UndoDoneDeletedTag    MessageID = "undo.done_deleted_tag" // "deleted tag %s"
	UndoDoneRemovedCommit MessageID = "undo.done_removed_commit"
	UndoResetFailed       MessageID = "undo.reset_failed" // "tag %s deleted, but removing the release commit failed (finish with `git reset --hard HEAD~1`): %w"

	// Stats command (cmd/stats.go): Short/Long text, the --repo/--prerelease
	// flag usage strings, and every literal in renderStats — the header,
	// section headers, row labels, the prerelease tag suffix, the "N older"
	// line, the "Latest release (%s)" line, and the footer note (PR6d).
	StatsShort MessageID = "stats.short"
	StatsLong  MessageID = "stats.long"

	StatsFlagRepoUsage       MessageID = "stats.flag_repo_usage"
	StatsFlagPrereleaseUsage MessageID = "stats.flag_prerelease_usage"

	StatsHeader               MessageID = "stats.header"
	StatsDownloadsSection     MessageID = "stats.downloads_section"
	StatsTotalLabel           MessageID = "stats.total_label"
	StatsLatestReleaseLabel   MessageID = "stats.latest_release_label"
	StatsReleasesSection      MessageID = "stats.releases_section"
	StatsPrereleaseSuffix     MessageID = "stats.prerelease_suffix"
	StatsOlderCount           MessageID = "stats.older_count"            // "… and %d older"
	StatsLatestReleaseSection MessageID = "stats.latest_release_section" // "Latest release (%s)"
	StatsGitHubSection        MessageID = "stats.github_section"
	StatsStarsLabel           MessageID = "stats.stars_label"
	StatsForksLabel           MessageID = "stats.forks_label"
	StatsFooterNote           MessageID = "stats.footer_note"

	// Auth command (cmd/auth.go): the group's and `status` subcommand's
	// Short/Long text, authStatus's not-authenticated and logged-in-as
	// lines, and the login/logout note commands' Short/body text (PR6d).
	AuthShort MessageID = "auth.short"
	AuthLong  MessageID = "auth.long"

	AuthStatusShort      MessageID = "auth.status_short"
	AuthNotAuthenticated MessageID = "auth.not_authenticated"
	AuthLoggedInAs       MessageID = "auth.logged_in_as" // "logged in as %s (via %s)"

	AuthLoginShort  MessageID = "auth.login_short"
	AuthLoginBody   MessageID = "auth.login_body"
	AuthLogoutShort MessageID = "auth.logout_short"
	AuthLogoutBody  MessageID = "auth.logout_body"

	// Image command (cmd/image.go): Short/Long text, the --version/--hash/
	// --upload/--link-only flag usage strings (--shape/--theme list literal
	// values the user types and stay untranslated, matching cmd/post.go's
	// --format convention), the shape/theme picker items (rebuilt as
	// shapeItems()/themeItems() call-time functions for the same reason
	// postFormatItems() was — a package var would freeze at "en" before
	// Execute() ever resolves the language), every pick.Run title, the
	// no-releases notice, the unknown-release error, the saved-file
	// confirmation, the save/link/both dest picker's item labels/descs, and
	// the still-saved-locally fallback note shown after an upload failure
	// (PR6d).
	ImageShort MessageID = "image.short"
	ImageLong  MessageID = "image.long"

	ImageFlagVersionUsage  MessageID = "image.flag_version_usage"
	ImageFlagHashUsage     MessageID = "image.flag_hash_usage"
	ImageFlagUploadUsage   MessageID = "image.flag_upload_usage"
	ImageFlagLinkOnlyUsage MessageID = "image.flag_link_only_usage"

	ImageShapeHorizontalLabel MessageID = "image.shape_horizontal_label"
	ImageShapeHorizontalDesc  MessageID = "image.shape_horizontal_desc"
	ImageShapeVerticalLabel   MessageID = "image.shape_vertical_label"
	ImageShapeVerticalDesc    MessageID = "image.shape_vertical_desc"
	ImageShapeSquareLabel     MessageID = "image.shape_square_label"
	ImageShapeSquareDesc      MessageID = "image.shape_square_desc"

	ImageThemeOrangeLabel MessageID = "image.theme_orange_label"
	ImageThemeOrangeDesc  MessageID = "image.theme_orange_desc"
	ImageThemeGreenLabel  MessageID = "image.theme_green_label"
	ImageThemeGreenDesc   MessageID = "image.theme_green_desc"
	ImageThemePurpleLabel MessageID = "image.theme_purple_label"
	ImageThemePurpleDesc  MessageID = "image.theme_purple_desc"

	ImagePickReleaseTitle MessageID = "image.pick_release_title"
	ImagePickShapeTitle   MessageID = "image.pick_shape_title"
	ImagePickThemeTitle   MessageID = "image.pick_theme_title"
	ImagePickDestTitle    MessageID = "image.pick_dest_title"

	ImageNoReleasesYet     MessageID = "image.no_releases_yet"
	ImageNoSuchRelease     MessageID = "image.no_such_release" // "no such release %q"
	ImageSaved             MessageID = "image.saved"           // "saved %s"
	ImageStillSavedLocally MessageID = "image.still_saved_locally"

	ImageDestBothLabel MessageID = "image.dest_both_label"
	ImageDestBothDesc  MessageID = "image.dest_both_desc" // "Write it to %s and upload for a QR + link"
	ImageDestSaveLabel MessageID = "image.dest_save_label"
	ImageDestSaveDesc  MessageID = "image.dest_save_desc" // "Write it to %s"
	ImageDestLinkLabel MessageID = "image.dest_link_label"
	ImageDestLinkDesc  MessageID = "image.dest_link_desc"
)
