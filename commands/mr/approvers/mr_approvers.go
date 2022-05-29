package approvers

import (
	"fmt"

	"github.com/profclems/glab/api"
	"github.com/profclems/glab/commands/mr/mrutils"

	"github.com/spf13/cobra"

	"github.com/profclems/glab/commands/cmdutils"
)

func NewCmdApprovers(f *cmdutils.Factory) *cobra.Command {
	mrApproversCmd := &cobra.Command{
		Use:     "approvers [<id> | <branch>] [flags]",
		Short:   `List merge request eligible approvers`,
		Long:    ``,
		Aliases: []string{},
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			apiClient, err := f.HttpClient()
			if err != nil {
				return err
			}

			mr, repo, err := mrutils.MRFromArgs(f, args, "any")
			if err != nil {
				return err
			}

			fmt.Fprintf(f.IO.StdOut, "\nListing Merge Request !%d eligible approvers\n", mr.IID)

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
