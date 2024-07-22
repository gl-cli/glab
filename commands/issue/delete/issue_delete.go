package delete

import (
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/commands/issue/issueutils"

	"github.com/spf13/cobra"
)

func NewCmdDelete(f *cmdutils.Factory) *cobra.Command {
	issueDeleteCmd := &cobra.Command{
		Use:     "delete <id>",
		Short:   `Delete an issue.`,
		Long:    ``,
		Aliases: []string{"del"},
		Example: heredoc.Doc(`
			glab issue delete 123
			glab issue del 123
			glab issue delete https://gitlab.com/profclems/glab/-/issues/123
		`),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			c := f.IO.Color()

			apiClient, err := f.HttpClient()
			if err != nil {
				return err
			}

			issues, repo, err := issueutils.IssuesFromArgs(apiClient, f.BaseRepo, args)
			if err != nil {
				return err
			}

			for _, issue := range issues {
				if f.IO.IsErrTTY && f.IO.IsaTTY {
					fmt.Fprintf(f.IO.StdErr, "- Deleting issue #%d.\n", issue.IID)
				}

				err := api.DeleteIssue(apiClient, repo.FullName(), issue.IID)
				if err != nil {
					return err
				}

				fmt.Fprintln(f.IO.StdErr, c.GreenCheck(), "Issue deleted.")
			}
			return nil
		},
	}
	return issueDeleteCmd
}
