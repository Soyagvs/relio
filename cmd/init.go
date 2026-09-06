package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/soyagvs/go-release/internal/config"
	"github.com/soyagvs/go-release/internal/gitrepo"
	"github.com/soyagvs/go-release/internal/ui"
)

func newInitCmd(f *releaseFlags) *cobra.Command {
	var project string

	c := &cobra.Command{
		Use:   "init",
		Short: "Create a .release.yaml in the current repository",
		Long:  "Write a .release.yaml with sensible defaults. The file holds configuration only — never secrets.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()

			repo, err := gitrepo.Open(f.dir)
			if err != nil {
				return fmt.Errorf("not a git repository — run `go-release init` inside a repo")
			}
			root := repo.Root()

			if config.Exists(root) {
				return fmt.Errorf("%s already exists at %s", config.FileName, root)
			}

			name := project
			if name == "" {
				name = guessProjectName(repo, root)
			}

			cfg := config.Default(name)
			if err := cfg.Save(root); err != nil {
				return err
			}

			fmt.Fprintln(out, ui.Banner(name, version))
			fmt.Fprintln(out)
			fmt.Fprintln(out, ui.Success([]string{filepath.Join(root, config.FileName) + " created"}))
			fmt.Fprintln(out)
			fmt.Fprintln(out, ui.Dim.Render("  Review it, commit it, then run `go-release`."))
			return nil
		},
	}

	c.Flags().StringVar(&project, "project", "", "project name (defaults to the repo/remote name)")
	return c
}

func guessProjectName(repo *gitrepo.Repo, root string) string {
	if url, err := repo.RemoteURL("origin"); err == nil && url != "" {
		return repoNameFromURL(url)
	}
	return filepath.Base(root)
}

func repoNameFromURL(url string) string {
	url = strings.TrimSpace(url)
	url = strings.TrimSuffix(url, ".git")
	url = strings.TrimSuffix(url, "/")
	if i := strings.LastIndexAny(url, "/:"); i >= 0 {
		return url[i+1:]
	}
	return url
}
