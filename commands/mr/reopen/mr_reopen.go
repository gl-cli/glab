package reopen

import (
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/commands/mr/mrutils"

	"github.com/spf13/cobra"
	"github.com/xanzy/go-gitlab"
)

func NewCmdReopen(f *cmdutils.Factory) *cobra.Command {
	mrReopenCmd := &cobra.Command{
		Use:   "reopen [<id>... | <branch>...]",
		Short: `Reopen a merge request.`,
		Example: heredoc.Doc(`
			$ glab mr reopen 123
			$ glab mr reopen 123 456 789
			$ glab mr reopen branch-1 branch-2

			# Uses the checked-out branch
			$ glab mr reopen
		`),
		Aliases: []string{"open"},
		RunE: func(cmd *cobra.Command, args []string) error {
			c := f.IO.Color()
			apiClient, err := f.HttpClient()
			if err != nil {
				return err
			}

			mrs, repo, err := mrutils.MRsFromArgs(f, args, "closed")
			if err != nil {
				return err
			}

			l := &gitlab.UpdateMergeRequestOptions{}
			l.StateEvent = gitlab.Ptr("reopen")
			for _, mr := range mrs {
				if err = mrutils.MRCheckErrors(mr, mrutils.MRCheckErrOptions{
					Opened: true,
					Merged: true,
				}); err != nil {
					return err
				}

				fmt.Fprintf(f.IO.StdOut, "- Reopening merge request !%d...\n", mr.IID)
				mr, err = api.UpdateMR(apiClient, repo.FullName(), mr.IID, l)
				if err != nil {
					return err
				}

				fmt.Fprintf(f.IO.StdOut, "%s Reopened merge request !%d.\n", c.GreenCheck(), mr.IID)
				fmt.Fprintln(f.IO.StdOut, mrutils.DisplayMR(f.IO.Color(), mr, f.IO.IsaTTY))
			}

			return nil
		},
	}

	return mrReopenCmd
}
