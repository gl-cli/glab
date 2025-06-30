package schedule

import (
	scheduleCreateCmd "gitlab.com/gitlab-org/cli/internal/commands/schedule/create"
	scheduleDeleteCmd "gitlab.com/gitlab-org/cli/internal/commands/schedule/delete"
	scheduleListCmd "gitlab.com/gitlab-org/cli/internal/commands/schedule/list"
	scheduleRunCmd "gitlab.com/gitlab-org/cli/internal/commands/schedule/run"
	scheduleUpdateCmd "gitlab.com/gitlab-org/cli/internal/commands/schedule/update"

	"gitlab.com/gitlab-org/cli/internal/cmdutils"

	"github.com/spf13/cobra"
)

func NewCmdSchedule(f cmdutils.Factory) *cobra.Command {
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
	scheduleCmd.AddCommand(scheduleUpdateCmd.NewCmdUpdate(f))

	return scheduleCmd
}
