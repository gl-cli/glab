package unsubscribe

import (
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/commands/mr/mrutils"

	"github.com/spf13/cobra"
)

var unsubscribeFromMR = func(client *gitlab.Client, projectID any, mrID int, opts gitlab.RequestOptionFunc) (*gitlab.MergeRequest, error) {
	mr, _, err := client.MergeRequests.UnsubscribeFromMergeRequest(projectID, mrID, opts)
	if err != nil {
		return nil, err
	}

	return mr, nil
}

func NewCmdUnsubscribe(f cmdutils.Factory) *cobra.Command {
	mrUnsubscribeCmd := &cobra.Command{
		Use:     "unsubscribe [<id> | <branch>]",
		Short:   `Unsubscribe from a merge request.`,
		Long:    ``,
		Aliases: []string{"unsub"},
		Example: heredoc.Doc(`
			Unsubscribe from a merge request
			- glab mr unsubscribe 123
			- glab mr unsub 123
			- glab mr unsubscribe branch

			Unsubscribe from multiple merge requests
			- glab mr unsubscribe 123 branch
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			c := f.IO().Color()

			apiClient, err := f.HttpClient()
			if err != nil {
				return err
			}

			mrs, repo, err := mrutils.MRsFromArgs(f, args, "any")
			if err != nil {
				return err
			}

			for _, mr := range mrs {
				if err = mrutils.MRCheckErrors(mr, mrutils.MRCheckErrOptions{
					Unsubscribed: true,
				}); err != nil {
					return err
				}

				fmt.Fprintf(f.IO().StdOut, "- Unsubscribing from merge request !%d.\n", mr.IID)

				mr, err = unsubscribeFromMR(apiClient, repo.FullName(), mr.IID, nil)
				if err != nil {
					return err
				}

				fmt.Fprintf(f.IO().StdOut, "%s You have successfully unsubscribed from merge request !%d.\n", c.RedCheck(), mr.IID)
				fmt.Fprintln(f.IO().StdOut, mrutils.DisplayMR(c, &mr.BasicMergeRequest, f.IO().IsaTTY))
			}

			return nil
		},
	}

	return mrUnsubscribeCmd
}
