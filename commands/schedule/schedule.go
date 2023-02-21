package schedule

import (
	scheduleListCmd "gitlab.com/gitlab-org/cli/commands/schedule/list"
	scheduleRunCmd "gitlab.com/gitlab-org/cli/commands/schedule/run"

	"gitlab.com/gitlab-org/cli/commands/cmdutils"

	"github.com/spf13/cobra"
)

func NewCmdSchedule(f *cmdutils.Factory) *cobra.Command {
	scheduleCmd := &cobra.Command{
		Use:     "schedule <command> [flags]",
		Short:   `Work with GitLab CI schedules`,
		Long:    ``,
		Aliases: []string{"sched", "skd"},
	}

	cmdutils.EnableRepoOverride(scheduleCmd, f)

	scheduleCmd.AddCommand(scheduleListCmd.NewCmdList(f))
	scheduleCmd.AddCommand(scheduleRunCmd.NewCmdRun(f))

	return scheduleCmd
}
