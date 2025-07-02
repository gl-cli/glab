package run

import (
	"fmt"
	"strconv"

	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/internal/iostreams"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
)

var runSchedule = func(client *gitlab.Client, repo string, schedule int, opts ...gitlab.RequestOptionFunc) error {
	_, err := client.PipelineSchedules.RunPipelineSchedule(repo, schedule, opts...)
	if err != nil {
		return fmt.Errorf("running scheduled pipeline status: %w", err)
	}

	return nil
}

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

	err = runSchedule(apiClient, repo.FullName(), o.scheduleID)
	if err != nil {
		return err
	}

	fmt.Fprintln(o.io.StdOut, "Started schedule with ID", o.scheduleID)

	return nil
}
