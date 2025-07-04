package close

import (
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"gitlab.com/gitlab-org/cli/internal/api"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/commands/mr/mrutils"

	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

func NewCmdClose(f cmdutils.Factory) *cobra.Command {
	mrCloseCmd := &cobra.Command{
		Use:   "close [<id> | <branch>]",
		Short: `Close a merge request.`,
		Long:  ``,
		Example: heredoc.Doc(`
			$ glab mr close 1

			# Close multiple merge requests at once
			$ glab mr close 1 2 3 4

			# Use the checked-out branch
			$ glab mr close

			$ glab mr close branch
			$ glab mr close username:branch
			$ glab mr close branch -R another/repo
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			c := f.IO().Color()
			apiClient, err := f.HttpClient()
			if err != nil {
				return err
			}

			mrs, repo, err := mrutils.MRsFromArgs(f, args, "opened")
			if err != nil {
				return err
			}

			l := &gitlab.UpdateMergeRequestOptions{}
			l.StateEvent = gitlab.Ptr("close")
			for _, mr := range mrs {
				if err = mrutils.MRCheckErrors(mr, mrutils.MRCheckErrOptions{
					Closed: true,
					Merged: true,
				}); err != nil {
					return err
				}
				fmt.Fprintf(f.IO().StdOut, "- Closing merge request...\n")
				_, err := api.UpdateMR(apiClient, repo.FullName(), mr.IID, l)
				if err != nil {
					return err
				}
				// Update the value of the merge request to closed so that mrutils.DisplayMR
				// prints it as red
				mr.State = "closed"

				fmt.Fprintf(f.IO().StdOut, "%s Closed merge request !%d.\n", c.RedCheck(), mr.IID)
				fmt.Fprintln(f.IO().StdOut, mrutils.DisplayMR(c, &mr.BasicMergeRequest, f.IO().IsaTTY))
			}

			return nil
		},
	}

	return mrCloseCmd
}
