package revoke

import (
	"fmt"

	"gitlab.com/gitlab-org/cli/internal/mcpannotations"

	"github.com/MakeNowJust/heredoc/v2"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/commands/mr/mrutils"

	"github.com/spf13/cobra"
)

func NewCmdRevoke(f cmdutils.Factory) *cobra.Command {
	mrRevokeCmd := &cobra.Command{
		Use:     "revoke [<id> | <branch>]",
		Short:   `Revoke approval on a merge request.`,
		Long:    ``,
		Aliases: []string{"unapprove"},
		Example: heredoc.Doc(`
		# Revoke approval on a merge request
		$ glab mr revoke 123
		$ glab mr unapprove 123
		$ glab mr revoke branch

		# Revoke approval on the currently checked out branch
		$ glab mr revoke
		# Revoke approval on merge request 123 on branch 456
		$ glab mr revoke 123 branch 456
		`),
		Annotations: map[string]string{
			mcpannotations.Destructive: "true",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			c := f.IO().Color()

			client, err := f.GitLabClient()
			if err != nil {
				return err
			}

			mrs, repo, err := mrutils.MRsFromArgs(f, args, "opened")
			if err != nil {
				return err
			}

			for _, mr := range mrs {
				if err = mrutils.MRCheckErrors(mr, mrutils.MRCheckErrOptions{
					Draft:  true,
					Closed: true,
					Merged: true,
				}); err != nil {
					return err
				}

				fmt.Fprintf(f.IO().StdOut, "- Revoking approval for merge request !%d...\n", mr.IID)

				_, err := client.MergeRequestApprovals.UnapproveMergeRequest(repo.FullName(), mr.IID)
				if err != nil {
					return err
				}

				fmt.Fprintln(f.IO().StdOut, c.GreenCheck(), "Merge request approval revoked.")
			}

			return nil
		},
	}

	return mrRevokeCmd
}
