package create

import (
	"fmt"

	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"

	"github.com/MakeNowJust/heredoc"
	"github.com/spf13/cobra"
	"github.com/xanzy/go-gitlab"
)

func NewCmdCreate(f *cmdutils.Factory) *cobra.Command {
	scheduleCreateCmd := &cobra.Command{
		Use:   "create [flags]",
		Short: `Schedule a new pipeline.`,
		Example: heredoc.Doc(`
			glab schedule create --cron "0 * * * *" --description "Describe your pipeline here" --ref "main"
		`),
		Long: ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			apiClient, err := f.HttpClient()
			if err != nil {
				return err
			}

			repo, err := f.BaseRepo()
			if err != nil {
				return err
			}

			l := &gitlab.CreatePipelineScheduleOptions{}

			description, _ := cmd.Flags().GetString("description")
			ref, _ := cmd.Flags().GetString("ref")
			cron, _ := cmd.Flags().GetString("cron")
			cronTimeZone, _ := cmd.Flags().GetString("cronTimeZone")
			active, _ := cmd.Flags().GetBool("active")

			l.Description = &description
			l.Ref = &ref
			l.Cron = &cron
			l.CronTimezone = &cronTimeZone
			l.Active = &active

			err = api.CreateSchedule(apiClient, repo.FullName(), l)
			if err != nil {
				return err
			}
			fmt.Fprintln(f.IO.StdOut, "Created schedule")

			return nil
		},
	}
	scheduleCreateCmd.Flags().String("description", "", "Description of the schedule")
	scheduleCreateCmd.Flags().String("ref", "", "Target branch or tag")
	scheduleCreateCmd.Flags().String("cron", "", "Cron interval pattern")
	scheduleCreateCmd.Flags().String("cronTimeZone", "UTC", "Cron timezone")
	scheduleCreateCmd.Flags().Bool("active", true, "Whether or not the schedule is active")

	_ = scheduleCreateCmd.MarkFlagRequired("ref")
	_ = scheduleCreateCmd.MarkFlagRequired("cron")
	_ = scheduleCreateCmd.MarkFlagRequired("description")

	return scheduleCreateCmd
}
