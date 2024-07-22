package revoke

import (
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/commands/mr/mrutils"

	"github.com/spf13/cobra"
)

func NewCmdRevoke(f *cmdutils.Factory) *cobra.Command {
	mrRevokeCmd := &cobra.Command{
		Use:     "revoke [<id> | <branch>]",
		Short:   `Revoke approval on a merge request.`,
		Long:    ``,
		Aliases: []string{"unapprove"},
		Example: heredoc.Doc(`
			$ glab mr revoke 123
			$ glab mr unapprove 123
			$ glab mr revoke branch

			# Uses the checked-out branch
			$ glab mr revoke

			$ glab mr revoke 123 branch 456
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			c := f.IO.Color()

			apiClient, err := f.HttpClient()
			if err != nil {
				return err
			}

			mrs, repo, err := mrutils.MRsFromArgs(f, args, "opened")
			if err != nil {
				return err
			}

			for _, mr := range mrs {
				if err = mrutils.MRCheckErrors(mr, mrutils.MRCheckErrOptions{
					WorkInProgress: true,
					Closed:         true,
					Merged:         true,
				}); err != nil {
					return err
				}

				fmt.Fprintf(f.IO.StdOut, "- Revoking approval for merge request !%d...\n", mr.IID)

				err = api.UnapproveMR(apiClient, repo.FullName(), mr.IID)
				if err != nil {
					return err
				}

				fmt.Fprintln(f.IO.StdOut, c.GreenCheck(), "Merge request approval revoked.")
			}

			return nil
		},
	}

	return mrRevokeCmd
}
