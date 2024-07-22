package delete

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

func NewCmdDelete(f *cmdutils.Factory) *cobra.Command {
	opts := &RunOpts{
		IO: f.IO,
	}
	scheduleDeleteCmd := &cobra.Command{
		Use:   "delete [flags]",
		Short: `Delete the schedule with the specified ID.`,
		Example: heredoc.Doc(`
			glab schedule delete 10
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

			scheduleId := int(id)
			if err != nil {
				return err
			}

			err = api.DeleteSchedule(apiClient, scheduleId, repo.FullName())
			if err != nil {
				return err
			}
			fmt.Fprintln(opts.IO.StdOut, "Deleted schedule with ID", scheduleId)

			return nil
		},
	}
	return scheduleDeleteCmd
}
