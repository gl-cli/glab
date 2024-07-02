package approvers

import (
	"fmt"

	"github.com/spf13/cobra"
	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/commands/mr/mrutils"
)

func NewCmdApprovers(f *cmdutils.Factory) *cobra.Command {
	mrApproversCmd := &cobra.Command{
		Use:     "approvers [<id> | <branch>] [flags]",
		Short:   `List eligible approvers for merge requests in any state.`,
		Long:    ``,
		Aliases: []string{},
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			apiClient, err := f.HttpClient()
			if err != nil {
				return err
			}

			// Obtain the MR from the positional arguments, but allow users to find approvers for
			// merge requests in any valid state
			mr, repo, err := mrutils.MRFromArgs(f, args, "any")
			if err != nil {
				return err
			}

			fmt.Fprintf(f.IO.StdOut, "\nListing merge request !%d eligible approvers:\n", mr.IID)

			mrApprovals, err := api.GetMRApprovalState(apiClient, repo.FullName(), mr.IID)
			if err != nil {
				return err
			}

			mrutils.PrintMRApprovalState(f.IO, mrApprovals)

			return nil
		},
	}

	return mrApproversCmd
}
