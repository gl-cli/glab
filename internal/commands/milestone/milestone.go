package milestone

import (
	"github.com/spf13/cobra"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	cmdCreate "gitlab.com/gitlab-org/cli/internal/commands/milestone/create"
	cmdGet "gitlab.com/gitlab-org/cli/internal/commands/milestone/get"
	cmdList "gitlab.com/gitlab-org/cli/internal/commands/milestone/list"
)

func NewCmdMilestone(f cmdutils.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "milestone <command>",
		Short: "Manage group or project milestones.",
		Long:  "",
	}

	cmdutils.EnableRepoOverride(cmd, f)

	cmd.AddCommand(cmdList.NewCmdList(f))
	cmd.AddCommand(cmdGet.NewCmdGet(f))
	cmd.AddCommand(cmdCreate.NewCmdCreate(f))

	return cmd
}
