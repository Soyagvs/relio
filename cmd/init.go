package cmd

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/soyagvs/relio/internal/config"
	"github.com/soyagvs/relio/internal/gitrepo"
	"github.com/soyagvs/relio/internal/ui"
)

func newInitCmd(f *releaseFlags) *cobra.Command {
	var project string

	c := &cobra.Command{
		Use:   "init",
		Short: "Create a .release.yaml in the current repository",
		Long:  "Write a .release.yaml with sensible defaults. The file holds configuration only — never secrets.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, err := gitrepo.Open(f.dir)
			if err != nil {
				return fmt.Errorf("not a git repository — run `relio init` inside a repo")
			}
			return runInit(cmd.OutOrStdout(), repo, project)
		},
	}

	c.Flags().StringVar(&project, "project", "", "project name (defaults to the repo/remote name)")
	return c
}

// runInit writes a fresh .release.yaml for repo and prints the confirmation.
// projectOverride wins when non-empty; otherwise the name is guessed from the
// repo/remote. Shared by `relio init` and the menu's Setup entry.
func runInit(w io.Writer, repo *gitrepo.Repo, projectOverride string) error {
	root := repo.Root()

	if config.Exists(root) {
		return fmt.Errorf("%s already exists at %s", config.FileName, root)
	}

	name := strings.TrimSpace(projectOverride)
	if name == "" {
		name = guessProjectName(repo, root)
	}

	cfg := config.Default(name)
	if err := cfg.Save(root); err != nil {
		return err
	}

	fmt.Fprintln(w, ui.Banner(name, version))
	fmt.Fprintln(w)
	fmt.Fprintln(w, ui.Success([]string{filepath.Join(root, config.FileName) + " created"}))
	fmt.Fprintln(w)
	fmt.Fprintln(w, ui.Dim.Render("  Review it, commit it, then run `relio`."))
	return nil
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
