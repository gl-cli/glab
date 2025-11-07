package milestone

import (
	"github.com/spf13/cobra"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	cmdCreate "gitlab.com/gitlab-org/cli/internal/commands/milestone/create"
	cmdDelete "gitlab.com/gitlab-org/cli/internal/commands/milestone/delete"
	cmdEdit "gitlab.com/gitlab-org/cli/internal/commands/milestone/edit"
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

	cmd.AddCommand(cmdCreate.NewCmdCreate(f))
	cmd.AddCommand(cmdDelete.NewCmdDelete(f))
	cmd.AddCommand(cmdEdit.NewCmdEdit(f))
	cmd.AddCommand(cmdGet.NewCmdGet(f))
	cmd.AddCommand(cmdList.NewCmdList(f))

	return cmd
}
