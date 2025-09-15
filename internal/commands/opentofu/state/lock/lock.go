package lock

import (
	"context"
	"fmt"

	"gitlab.com/gitlab-org/cli/internal/mcpannotations"

	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/internal/iostreams"
)

type options struct {
	io           *iostreams.IOStreams
	baseRepo     func() (glrepo.Interface, error)
	gitlabClient func() (*gitlab.Client, error)

	stateName string
}

func NewCmd(f cmdutils.Factory) *cobra.Command {
	opts := &options{
		io:           f.IO(),
		baseRepo:     f.BaseRepo,
		gitlabClient: f.GitLabClient,
	}

	cmd := &cobra.Command{
		Use:   "lock <state>",
		Short: `Lock the given state.`,
		Args:  cobra.ExactArgs(1),
		Annotations: map[string]string{
			mcpannotations.Destructive: "true",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.complete(args)
			return opts.run(cmd.Context())
		},
	}

	return cmd
}

func (o *options) complete(args []string) {
	o.stateName = args[0]
}

func (o *options) run(ctx context.Context) error {
	repo, err := o.baseRepo()
	if err != nil {
		return err
	}

	client, err := o.gitlabClient()
	if err != nil {
		return err
	}

	if _, err := client.TerraformStates.Lock(repo.FullName(), o.stateName, gitlab.WithContext(ctx)); err != nil {
		return err
	}

	fmt.Fprintf(o.io.StdOut, "Locked state %s\n", o.stateName)
	return nil
}
