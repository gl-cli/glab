package rebase

import (
	"github.com/MakeNowJust/heredoc/v2"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/commands/mr/mrutils"
	"gitlab.com/gitlab-org/cli/internal/iostreams"

	"github.com/spf13/cobra"
)

type options struct {
	f          cmdutils.Factory // TODO: refactor mrutils to not rely on factory
	io         *iostreams.IOStreams
	httpClient func() (*gitlab.Client, error)

	// SkipCI: rebase merge request while skipping CI/CD pipeline.
	SkipCI bool

	args []string
}

func NewCmdRebase(f cmdutils.Factory) *cobra.Command {
	opts := &options{
		f:          f,
		io:         f.IO(),
		httpClient: f.HttpClient,
	}

	mrRebaseCmd := &cobra.Command{
		Use:   "rebase [<id> | <branch>] [flags]",
		Short: `Rebase the source branch of a merge request against its target branch.`,
		Long: heredoc.Doc(`If you don't have permission to push to the merge request's source branch, you'll get a 403 Forbidden response.
		`),
		Example: heredoc.Doc(`
			# Rebase merge request 123
			$ glab mr rebase 123

			# Rebase current branch
			$ glab mr rebase

			# Rebase merge request from branch
			$ glab mr rebase branch
			$ glab mr rebase branch --skip-ci
		`),
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.complete(args)

			return opts.run()
		},
	}
	mrRebaseCmd.Flags().BoolVarP(&opts.SkipCI, "skip-ci", "", false, "Rebase merge request while skipping CI/CD pipeline.")

	return mrRebaseCmd
}

func (o *options) complete(args []string) {
	o.args = args
}

func (o *options) run() error {
	apiClient, err := o.httpClient()
	if err != nil {
		return err
	}

	mr, repo, err := mrutils.MRFromArgs(o.f, o.args, "opened")
	if err != nil {
		return err
	}

	return mrutils.RebaseMR(
		o.io,
		apiClient,
		repo,
		mr,
		&gitlab.RebaseMergeRequestOptions{SkipCI: gitlab.Ptr(o.SkipCI)},
	)
}
