package cmd

import (
	"fmt"
	"strings"

	"github.com/soyagvs/relio/internal/i18n"
	"github.com/soyagvs/relio/internal/ui"
)

// reference is the full command + flag listing shown by the menu's Help entry.
type refRow struct{ name, desc string }

// helpTables builds the reference rows at call time — never as package-level
// vars — so every desc resolves through i18n.T() against whatever language
// is active when helpReference() is invoked, not whatever was active at
// package init (which is always "en", before Execute() ever runs). Row
// *names* (command syntax, flag syntax) stay literal English on purpose:
// they are code the user types verbatim, not prose. Menu-row names reuse
// internal/menu's own i18n keys instead of duplicating them.
func helpTables() (commands, releaseFlags, postFlags, imageFlags, menu []refRow) {
	commands = []refRow{
		{"relio", i18n.T(i18n.HelpCmdRelioDesc)},
		{"relio status", i18n.T(i18n.HelpCmdStatusDesc)},
		{"relio check", i18n.T(i18n.HelpCmdCheckDesc)},
		{"relio guide", i18n.T(i18n.HelpCmdGuideDesc)},
		{"relio stats", i18n.T(i18n.HelpCmdStatsDesc)},
		{"relio init", i18n.T(i18n.HelpCmdInitDesc)},
		{"relio post", i18n.T(i18n.HelpCmdPostDesc)},
		{"relio image", i18n.T(i18n.HelpCmdImageDesc)},
		{"relio auth", i18n.T(i18n.HelpCmdAuthDesc)},
		{"relio version", i18n.T(i18n.VersionShort)},
	}
	releaseFlags = []refRow{
		{"--patch / --minor / --major", i18n.T(i18n.HelpFlagBumpDesc)},
		{"-y, --yes", i18n.T(i18n.HelpFlagYesDesc)},
		{"--no-changelog", i18n.T(i18n.HelpFlagNoChangelogDesc)},
		{"--no-tag", i18n.T(i18n.HelpFlagNoTagDesc)},
		{"--rc", i18n.T(i18n.HelpFlagRCDesc)},
		{"--publish", i18n.T(i18n.HelpFlagPublishDesc)},
		{"--no-hash", i18n.T(i18n.HelpFlagNoHashDesc)},
		{"-C, --dir <path>", i18n.T(i18n.HelpFlagDirDesc)},
	}
	postFlags = []refRow{
		{"--format minimal", i18n.T(i18n.HelpPostMinimalDesc)},
		{"--format social", i18n.T(i18n.HelpPostSocialDesc)},
		{"--format technical", i18n.T(i18n.HelpPostTechnicalDesc)},
		{"--format casual", i18n.T(i18n.HelpPostCasualDesc)},
		{"--format changelog", i18n.T(i18n.HelpPostChangelogDesc)},
	}
	imageFlags = []refRow{
		{"--shape", i18n.T(i18n.HelpImageShapeDesc)},
		{"--theme", i18n.T(i18n.HelpImageThemeDesc)},
		{"--hash", i18n.T(i18n.HelpImageHashDesc)},
		{"--upload", i18n.T(i18n.HelpImageUploadDesc)},
		{"--link-only", i18n.T(i18n.HelpImageLinkOnlyDesc)},
	}
	menu = []refRow{
		{i18n.T(i18n.MenuReleaseLabel), i18n.T(i18n.HelpMenuReleaseDesc)},
		{i18n.T(i18n.MenuStatusLabel), i18n.T(i18n.HelpMenuStatusDesc)},
		{i18n.T(i18n.MenuCheckLabel), i18n.T(i18n.HelpMenuCheckDesc)},
		{i18n.T(i18n.MenuViewReleasesLabel), i18n.T(i18n.HelpMenuReleasesDesc)},
		{i18n.T(i18n.MenuReleaseTextLabel), i18n.T(i18n.HelpMenuAnnouncementDesc)},
		{i18n.T(i18n.MenuReleaseImageLabel), i18n.T(i18n.HelpMenuReleaseImageDesc)},
		{i18n.T(i18n.MenuAuthLabel), i18n.T(i18n.HelpMenuAuthDesc)},
		{i18n.T(i18n.MenuSetupLabel), i18n.T(i18n.HelpMenuSetupDesc)},
		{i18n.T(i18n.MenuGuideLabel), i18n.T(i18n.HelpMenuGuideDesc)},
		{i18n.T(i18n.MenuHelpLabel), i18n.T(i18n.HelpMenuHelpDesc)},
		{i18n.T(i18n.MenuExitLabel), i18n.T(i18n.MenuExitDesc)},
	}
	return
}

func helpReference() string {
	var b strings.Builder
	b.WriteString(ui.Banner("", version) + "\n\n")

	section := func(title string, rows []refRow) {
		b.WriteString(ui.Key.Render(strings.ToUpper(title)) + "\n")
		width := 0
		for _, r := range rows {
			if len(r.name) > width {
				width = len(r.name)
			}
		}
		for _, r := range rows {
			b.WriteString("  " + fmt.Sprintf("%-*s", width, r.name) + "  " + ui.Dim.Render(r.desc) + "\n")
		}
		b.WriteString("\n")
	}

	commands, releaseFlags, postFlags, imageFlags, menu := helpTables()
	section(i18n.T(i18n.HelpSectionCommands), commands)
	section(i18n.T(i18n.HelpSectionReleaseFlags), releaseFlags)
	section(i18n.T(i18n.HelpSectionPostFlags), postFlags)
	section(i18n.T(i18n.HelpSectionImageFlags), imageFlags)
	section(i18n.T(i18n.HelpSectionMenu), menu)

	b.WriteString(ui.Dim.Render(i18n.T(i18n.HelpFooter)))
	return b.String()
}
