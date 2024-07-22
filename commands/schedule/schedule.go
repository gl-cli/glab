package schedule

import (
	scheduleCreateCmd "gitlab.com/gitlab-org/cli/commands/schedule/create"
	scheduleDeleteCmd "gitlab.com/gitlab-org/cli/commands/schedule/delete"
	scheduleListCmd "gitlab.com/gitlab-org/cli/commands/schedule/list"
	scheduleRunCmd "gitlab.com/gitlab-org/cli/commands/schedule/run"

	"gitlab.com/gitlab-org/cli/commands/cmdutils"

	"github.com/spf13/cobra"
)

func NewCmdSchedule(f *cmdutils.Factory) *cobra.Command {
	scheduleCmd := &cobra.Command{
		Use:     "schedule <command> [flags]",
		Short:   `Work with GitLab CI/CD schedules.`,
		Long:    ``,
		Aliases: []string{"sched", "skd"},
	}

	cmdutils.EnableRepoOverride(scheduleCmd, f)

	scheduleCmd.AddCommand(scheduleListCmd.NewCmdList(f))
	scheduleCmd.AddCommand(scheduleRunCmd.NewCmdRun(f))
	scheduleCmd.AddCommand(scheduleCreateCmd.NewCmdCreate(f))
	scheduleCmd.AddCommand(scheduleDeleteCmd.NewCmdDelete(f))

	return scheduleCmd
}
