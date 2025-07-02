package list

import (
	"fmt"

	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/commands/ci/ciutils"
	"gitlab.com/gitlab-org/cli/internal/utils"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

var getSchedules = func(client *gitlab.Client, l *gitlab.ListPipelineSchedulesOptions, repo string) ([]*gitlab.PipelineSchedule, error) {
	schedules, _, err := client.PipelineSchedules.ListPipelineSchedules(repo, l)
	return schedules, err
}

func NewCmdList(f cmdutils.Factory) *cobra.Command {
	scheduleListCmd := &cobra.Command{
		Use:   "list [flags]",
		Short: `Get the list of schedules.`,
		Example: heredoc.Doc(`
			# List all scheduled pipelines
			$ glab schedule list
			> Showing schedules for project gitlab-org/cli
			> ID  Description                    Cron            Ref    Active
			> 1   Daily build                   0 0 * * *       main   true
			> 2   Weekly deployment             0 0 * * 0       main   true
		`),
		Long: ``,
		Args: cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			apiClient, err := f.HttpClient()
			if err != nil {
				return err
			}

			repo, err := f.BaseRepo()
			if err != nil {
				return err
			}

			l := &gitlab.ListPipelineSchedulesOptions{}
			page, _ := cmd.Flags().GetInt("page")
			l.Page = page
			perPage, _ := cmd.Flags().GetInt("per-page")
			l.PerPage = perPage

			schedules, err := getSchedules(apiClient, l, repo.FullName())
			if err != nil {
				return err
			}

			title := utils.NewListTitle("schedule")
			title.RepoName = repo.FullName()
			title.Page = l.Page
			title.CurrentPageTotal = len(schedules)

			fmt.Fprintf(f.IO().StdOut, "%s\n%s\n", title.Describe(), ciutils.DisplaySchedules(f.IO(), schedules, repo.FullName()))
			return nil
		},
	}
	scheduleListCmd.Flags().IntP("page", "p", 1, "Page number.")
	scheduleListCmd.Flags().IntP("per-page", "P", 30, "Number of items to list per page.")

	return scheduleListCmd
}
