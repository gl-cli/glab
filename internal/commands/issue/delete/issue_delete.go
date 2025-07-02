package delete

import (
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/commands/issue/issueutils"

	"github.com/spf13/cobra"
)

var deleteIssue = func(client *gitlab.Client, projectID any, issueID int) error {
	_, err := client.Issues.DeleteIssue(projectID, issueID)
	if err != nil {
		return err
	}

	return nil
}

func NewCmdDelete(f cmdutils.Factory) *cobra.Command {
	issueDeleteCmd := &cobra.Command{
		Use:     "delete <id>",
		Short:   `Delete an issue.`,
		Long:    ``,
		Aliases: []string{"del"},
		Example: heredoc.Doc(`
			$ glab issue delete 123
			$ glab issue del 123
			$ glab issue delete https://gitlab.com/profclems/glab/-/issues/123
		`),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			c := f.IO().Color()

			gitlabClient, err := f.HttpClient()
			if err != nil {
				return err
			}

			issues, repo, err := issueutils.IssuesFromArgs(f.ApiClient, gitlabClient, f.BaseRepo, f.DefaultHostname(), args)
			if err != nil {
				return err
			}

			for _, issue := range issues {
				if f.IO().IsErrTTY && f.IO().IsaTTY {
					fmt.Fprintf(f.IO().StdErr, "- Deleting issue #%d.\n", issue.IID)
				}

				err := deleteIssue(gitlabClient, repo.FullName(), issue.IID)
				if err != nil {
					return err
				}

				fmt.Fprintln(f.IO().StdErr, c.GreenCheck(), "Issue deleted.")
			}
			return nil
		},
	}
	return issueDeleteCmd
}
