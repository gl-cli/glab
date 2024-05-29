package run

import (
	"fmt"
	"strconv"

	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/pkg/iostreams"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
)

type RunOpts struct {
	ScheduleId int
	IO         *iostreams.IOStreams
}

func NewCmdRun(f *cmdutils.Factory) *cobra.Command {
	opts := &RunOpts{
		IO: f.IO,
	}
	scheduleRunCmd := &cobra.Command{
		Use:   "run <id>",
		Short: `Run the specified scheduled pipeline.`,
		Example: heredoc.Doc(`
			glab schedule run 1
		`),
		Long: ``,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			apiClient, err := f.HttpClient()
			if err != nil {
				return err
			}

			repo, err := f.BaseRepo()
			if err != nil {
				return err
			}

			id, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return err
			}

			opts.ScheduleId = int(id)
			if err != nil {
				return err
			}

			err = api.RunSchedule(apiClient, repo.FullName(), opts.ScheduleId)
			if err != nil {
				return err
			}

			fmt.Fprintln(opts.IO.StdOut, "Started schedule with ID", opts.ScheduleId)

			return nil
		},
	}
	return scheduleRunCmd
}
