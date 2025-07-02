package list

import (
	"encoding/json"
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/cli/internal/api"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
)

func NewCmdList(f cmdutils.Factory) *cobra.Command {
	securefileListCmd := &cobra.Command{
		Use:     "list [flags]",
		Short:   `List secure files for a project.`,
		Long:    ``,
		Aliases: []string{"ls"},
		Example: heredoc.Doc(`
			List all secure files.
			- glab securefile list

			List all secure files with 'cmd' alias.
			- glab securefile ls

			List a specific page of secure files.
			- glab securefile list --page 2

			List a specific page of secure files, with a custom page size.
			- glab securefile list --page 2 --per-page 10
		`),
		Args: cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			apiClient, err := f.HttpClient()
			if err != nil {
				return err
			}

			repo, err := f.BaseRepo()
			if err != nil {
				return err
			}

			l := &gitlab.ListProjectSecureFilesOptions{
				Page:    1,
				PerPage: api.DefaultListLimit,
			}

			if p, _ := cmd.Flags().GetInt("page"); p != 0 {
				l.Page = p
			}

			if p, _ := cmd.Flags().GetInt("per-page"); p != 0 {
				l.PerPage = p
			}

			files, _, err := apiClient.SecureFiles.ListProjectSecureFiles(repo.FullName(), l)
			if err != nil {
				return fmt.Errorf("Error listing secure files: %v", err)
			}

			fileListJSON, _ := json.Marshal(files)
			fmt.Fprintln(f.IO().StdOut, string(fileListJSON))
			return nil
		},
	}

	securefileListCmd.Flags().IntP("page", "p", 1, "Page number.")
	securefileListCmd.Flags().IntP("per-page", "P", 30, "Number of items to list per page.")

	return securefileListCmd
}
