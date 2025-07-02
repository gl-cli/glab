package subscribe

import (
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/commands/mr/mrutils"

	"github.com/spf13/cobra"
)

var subscribeToMR = func(client *gitlab.Client, projectID any, mrID int, opts gitlab.RequestOptionFunc) (*gitlab.MergeRequest, error) {
	mr, _, err := client.MergeRequests.SubscribeToMergeRequest(projectID, mrID, opts)
	if err != nil {
		return nil, err
	}

	return mr, nil
}

func NewCmdSubscribe(f cmdutils.Factory) *cobra.Command {
	mrSubscribeCmd := &cobra.Command{
		Use:     "subscribe [<id> | <branch>]",
		Short:   `Subscribe to a merge request.`,
		Long:    ``,
		Aliases: []string{"sub"},
		Example: heredoc.Doc(`
			Subscribe to a merge request
			- glab mr subscribe 123
			- glab mr sub 123
			- glab mr subscribe branch

			Subscribe to multiple merge requests
			- glab mr subscribe 123 branch
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
					Subscribed: true,
				}); err != nil {
					return err
				}

				fmt.Fprintf(f.IO().StdOut, "- Subscribing to merge request !%d.\n", mr.IID)

				mr, err = subscribeToMR(apiClient, repo.FullName(), mr.IID, nil)
				if err != nil {
					return err
				}

				fmt.Fprintf(f.IO().StdOut, "%s You have successfully subscribed to merge request !%d.\n", c.GreenCheck(), mr.IID)
				fmt.Fprintln(f.IO().StdOut, mrutils.DisplayMR(c, &mr.BasicMergeRequest, f.IO().IsaTTY))
			}

			return nil
		},
	}

	return mrSubscribeCmd
}
