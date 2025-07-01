package delete

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

type options struct {
	scheduleID int

	io         *iostreams.IOStreams
	httpClient func() (*gitlab.Client, error)
	baseRepo   func() (glrepo.Interface, error)
}

func NewCmdDelete(f cmdutils.Factory) *cobra.Command {
	opts := &options{
		io:         f.IO(),
		httpClient: f.HttpClient,
		baseRepo:   f.BaseRepo,
	}
	scheduleDeleteCmd := &cobra.Command{
		Use:   "delete <id> [flags]",
		Short: `Delete the schedule with the specified ID.`,
		Example: heredoc.Doc(`
			# Delete a scheduled pipeline with ID 10
			$ glab schedule delete 10
			> Deleted schedule with ID 10
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
	return scheduleDeleteCmd
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

	_, err = apiClient.PipelineSchedules.DeletePipelineSchedule(repo.FullName(), o.scheduleID)
	if err != nil {
		return err
	}
	fmt.Fprintln(o.io.StdOut, "Deleted schedule with ID", o.scheduleID)

	return nil
}
