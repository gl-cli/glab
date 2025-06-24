package run

import (
	"fmt"
	"strconv"

	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/pkg/iostreams"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
)

type options struct {
	scheduleID int

	io         *iostreams.IOStreams
	httpClient func() (*gitlab.Client, error)
	baseRepo   func() (glrepo.Interface, error)
}

func NewCmdRun(f cmdutils.Factory) *cobra.Command {
	opts := &options{
		io:         f.IO(),
		httpClient: f.HttpClient,
		baseRepo:   f.BaseRepo,
	}
	scheduleRunCmd := &cobra.Command{
		Use:   "run <id>",
		Short: `Run the specified scheduled pipeline.`,
		Example: heredoc.Doc(`
			# Run a scheduled pipeline with ID 1
			$ glab schedule run 1
			> Started schedule with ID 1
		`),
		Long: ``,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := opts.complete(args); err != nil {
				return err
			}

			return opts.run()
		},
	}
	return scheduleRunCmd
}

func (o *options) complete(args []string) error {
	id, err := strconv.ParseUint(args[0], 10, 64)
	if err != nil {
		return err
	}
	o.scheduleID = int(id)

	return nil
}

func (o *options) run() error {
	apiClient, err := o.httpClient()
	if err != nil {
		return err
	}

	repo, err := o.baseRepo()
	if err != nil {
		return err
	}

	err = api.RunSchedule(apiClient, repo.FullName(), o.scheduleID)
	if err != nil {
		return err
	}

	fmt.Fprintln(o.io.StdOut, "Started schedule with ID", o.scheduleID)

	return nil
}
