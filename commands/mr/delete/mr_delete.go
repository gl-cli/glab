package delete

import (
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/commands/mr/mrutils"

	"github.com/spf13/cobra"
)

func NewCmdDelete(f *cmdutils.Factory) *cobra.Command {
	mrDeleteCmd := &cobra.Command{
		Use:     "delete [<id> | <branch>]",
		Short:   `Delete a merge request.`,
		Long:    ``,
		Aliases: []string{"del"},
		Example: heredoc.Doc(`
			$ glab mr delete 123

			# Delete multiple merge requests by ID and branch name
			$ glab mr delete 123 branch-name 789

			# Delete merge requests !1, !2, !3, !4, !5
			$ glab mr delete 1,2,branch-related-to-mr-3,4,5

			$ glab mr del 123
			$ glab mr delete branch
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := f.IO.Color()
			apiClient, err := f.HttpClient()
			if err != nil {
				return err
			}

			mrs, repo, err := mrutils.MRsFromArgs(f, args, "any")
			if err != nil {
				return err
			}

			for _, mr := range mrs {
				fmt.Fprintf(f.IO.StdOut, "- Deleting merge request !%d.\n", mr.IID)
				if err = api.DeleteMR(apiClient, repo.FullName(), mr.IID); err != nil {
					return err
				}
				fmt.Fprintf(f.IO.StdOut, "%s Merge request !%d deleted.\n", c.RedCheck(), mr.IID)
			}

			return nil
		},
	}

	return mrDeleteCmd
}
