package cmd

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/soyagvs/relio/internal/ghstats"
	"github.com/soyagvs/relio/internal/i18n"
	"github.com/soyagvs/relio/internal/ui"
)

// defaultRepo is Relio's own repository — `relio stats` shows Relio's numbers.
const defaultRepo = "soyagvs/relio"

func newStatsCmd() *cobra.Command {
	var repo string
	var pre bool

	c := &cobra.Command{
		Use:   "stats",
		Short: i18n.T(i18n.StatsShort),
		Long:  i18n.T(i18n.StatsLong),
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := ghstats.Fetch(cmd.Context(), nil, repo)
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), renderStats(s, pre))
			return nil
		},
	}

	c.Flags().StringVar(&repo, "repo", defaultRepo, i18n.T(i18n.StatsFlagRepoUsage))
	c.Flags().BoolVar(&pre, "prerelease", false, i18n.T(i18n.StatsFlagPrereleaseUsage))
	return c
}

func renderStats(s *ghstats.Stats, includePre bool) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n\n", ui.Key.Render(i18n.T(i18n.StatsHeader)))

	row := func(label, val string) {
		fmt.Fprintf(&b, "  %s %s\n", ui.Dim.Render(fmt.Sprintf("%-18s", label)), val)
	}

	latestDownloads := 0
	if s.Latest != nil {
		latestDownloads = s.Latest.Downloads
	}

	fmt.Fprintln(&b, ui.Key.Render(i18n.T(i18n.StatsDownloadsSection)))
	row(i18n.T(i18n.StatsTotalLabel), num(s.TotalDownloads))
	row(i18n.T(i18n.StatsLatestReleaseLabel), num(latestDownloads))

	// per-release list
	rels := make([]ghstats.Release, 0, len(s.Releases))
	for _, r := range s.Releases {
		if r.Prerelease && !includePre {
			continue
		}
		rels = append(rels, r)
	}
	if len(rels) > 0 {
		fmt.Fprintf(&b, "\n%s\n", ui.Key.Render(i18n.T(i18n.StatsReleasesSection)))

		const maxRows = 12
		older := 0
		if len(rels) > maxRows {
			older = len(rels) - maxRows
			rels = rels[:maxRows]
		}

		w := 0
		for _, r := range rels {
			if len(r.Tag) > w {
				w = len(r.Tag)
			}
		}
		for _, r := range rels {
			tag := r.Tag
			if r.Prerelease {
				tag += i18n.T(i18n.StatsPrereleaseSuffix)
			}
			fmt.Fprintf(&b, "  %s %s\n", ui.Dim.Render(fmt.Sprintf("%-*s", w+6, tag)), num(r.Downloads))
		}
		if older > 0 {
			fmt.Fprintf(&b, "  %s\n", ui.Dim.Render(i18n.T(i18n.StatsOlderCount, older)))
		}
	}

	// per-platform breakdown for the latest release
	if s.Latest != nil {
		type pc struct {
			plat string
			n    int
		}
		var pcs []pc
		for _, a := range s.Latest.Assets {
			if a.Platform == "" {
				continue
			}
			pcs = append(pcs, pc{a.Platform, a.DownloadCount})
		}
		sort.Slice(pcs, func(i, j int) bool {
			if pcs[i].n != pcs[j].n {
				return pcs[i].n > pcs[j].n
			}
			return pcs[i].plat < pcs[j].plat
		})
		if len(pcs) > 0 {
			fmt.Fprintf(&b, "\n%s\n", ui.Key.Render(i18n.T(i18n.StatsLatestReleaseSection, s.Latest.Tag)))
			w := 0
			for _, p := range pcs {
				if len(p.plat) > w {
					w = len(p.plat)
				}
			}
			for _, p := range pcs {
				fmt.Fprintf(&b, "  %s %s\n", ui.Dim.Render(fmt.Sprintf("%-*s", w+2, p.plat)), num(p.n))
			}
		}
	}

	fmt.Fprintf(&b, "\n%s\n", ui.Key.Render(i18n.T(i18n.StatsGitHubSection)))
	row(i18n.T(i18n.StatsStarsLabel), num(s.Stars))
	row(i18n.T(i18n.StatsForksLabel), num(s.Forks))

	fmt.Fprintf(&b, "\n%s\n", ui.Dim.Render(i18n.T(i18n.StatsFooterNote)))
	return b.String()
}

// num formats an int with thousands separators: 1284 -> "1,284".
func num(n int) string {
	s := strconv.Itoa(n)
	neg := strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(s, "-")
	var parts []string
	for len(s) > 3 {
		parts = append([]string{s[len(s)-3:]}, parts...)
		s = s[:len(s)-3]
	}
	parts = append([]string{s}, parts...)
	out := strings.Join(parts, ",")
	if neg {
		out = "-" + out
	}
	return out
}
