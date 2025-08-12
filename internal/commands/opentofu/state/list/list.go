package list

import (
	"fmt"

	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/internal/iostreams"
	"gitlab.com/gitlab-org/cli/internal/tableprinter"
)

type options struct {
	io           *iostreams.IOStreams
	baseRepo     func() (glrepo.Interface, error)
	gitlabClient func() (*gitlab.Client, error)
}

func NewCmd(f cmdutils.Factory) *cobra.Command {
	opts := &options{
		io:           f.IO(),
		baseRepo:     f.BaseRepo,
		gitlabClient: f.GitLabClient,
	}

	cmd := &cobra.Command{
		Use:   "list [flags]",
		Short: `List states`,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return opts.run()
		},
	}

	return cmd
}

func (o *options) run() error {
	repo, err := o.baseRepo()
	if err != nil {
		return err
	}

	client, err := o.gitlabClient()
	if err != nil {
		return err
	}

	states, _, err := client.TerraformStates.List(repo.FullName())
	if err != nil {
		return err
	}

	c := o.io.Color()
	table := tableprinter.NewTablePrinter()
	table.AddRow(c.Bold("Name"), c.Bold("Latest Version Serial"), c.Bold("Created At"), c.Bold("Updated At"), c.Bold("Locked At"))
	for _, state := range states {
		table.AddRow(state.Name, state.LatestVersion.Serial, state.CreatedAt, state.UpdatedAt, state.LockedAt)
	}
	fmt.Fprint(o.io.StdOut, table.Render())
	return nil
}
