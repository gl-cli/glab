package issues

import (
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/commands/issue/issueutils"
	"gitlab.com/gitlab-org/cli/commands/mr/mrutils"
	"gitlab.com/gitlab-org/cli/pkg/utils"

	"github.com/xanzy/go-gitlab"
)

func NewCmdIssues(f *cmdutils.Factory) *cobra.Command {
	mrIssuesCmd := &cobra.Command{
		Use:     "issues [<id> | <branch>]",
		Short:   `Get issues related to a particular merge request.`,
		Long:    ``,
		Aliases: []string{"issue"},
		Args:    cobra.MaximumNArgs(1),
		Example: heredoc.Doc(`
			$ glab mr issues 46
			$ glab mr issues branch

			# Use the checked-out branch
			$ glab mr issues
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error

			apiClient, err := f.HttpClient()
			if err != nil {
				return err
			}

			mr, repo, err := mrutils.MRFromArgs(f, args, "any")
			if err != nil {
				return err
			}

			l := &gitlab.GetIssuesClosedOnMergeOptions{}

			mrIssues, err := api.GetMRLinkedIssues(apiClient, repo.FullName(), mr.IID, l)
			if err != nil {
				return err
			}

			title := utils.NewListTitle("issue")
			title.RepoName = repo.FullName()
			title.Page = 0
			title.ListActionType = "search"
			title.CurrentPageTotal = len(mrIssues)

			fmt.Fprintf(f.IO.StdOut, "%s\n%s\n", title.Describe(), issueutils.DisplayIssueList(f.IO, mrIssues, repo.FullName()))
			return nil
		},
	}

	return mrIssuesCmd
}
