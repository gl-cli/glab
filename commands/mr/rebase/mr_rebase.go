package rebase

import (
	"github.com/MakeNowJust/heredoc"
	"github.com/xanzy/go-gitlab"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/commands/mr/mrutils"

	"github.com/spf13/cobra"
)

type MRRebaseOptions struct {
	// SkipCI: rebase merge request while skipping CI/CD pipeline.
	SkipCI bool
}

func NewCmdRebase(f *cmdutils.Factory) *cobra.Command {
	opts := &MRRebaseOptions{}

	mrRebaseCmd := &cobra.Command{
		Use:   "rebase [<id> | <branch>] [flags]",
		Short: `Automatically rebase the source_branch of the merge request against its target_branch.`,
		Long: heredoc.Doc(`If you don't have permissions to push to the merge request's source branch - you'll get a 403 Forbidden response.
		`),
		Example: heredoc.Doc(`
			glab mr rebase 123
			glab mr rebase  # get from current branch
			glab mr rebase branch
			glab mr rebase branch --skip-ci
		`),
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error

			apiClient, err := f.HttpClient()
			if err != nil {
				return err
			}

			mr, repo, err := mrutils.MRFromArgs(f, args, "opened")
			if err != nil {
				return err
			}

			if err = mrutils.RebaseMR(f.IO, apiClient, repo, mr,
				&gitlab.RebaseMergeRequestOptions{
					SkipCI: gitlab.Ptr(opts.SkipCI),
				}); err != nil {
				return err
			}

			return nil
		},
	}

	mrRebaseCmd.Flags().BoolVarP(
		&opts.SkipCI,
		"skip-ci",
		"",
		false,
		"Rebase merge request while skipping CI/CD pipeline.",
	)

	return mrRebaseCmd
}
