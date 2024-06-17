package stack

import (
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	stackCreateCmd "gitlab.com/gitlab-org/cli/commands/stack/create"
	stackMoveCmd "gitlab.com/gitlab-org/cli/commands/stack/navigate"
	stackSaveCmd "gitlab.com/gitlab-org/cli/commands/stack/save"
	stackSyncCmd "gitlab.com/gitlab-org/cli/commands/stack/sync"

	"github.com/spf13/cobra"
)

func NewCmdStack(f *cmdutils.Factory) *cobra.Command {
	stackCmd := &cobra.Command{
		Use:     "stack <command> [flags]",
		Short:   `Work with stacked diffs.`,
		Long:    ``,
		Aliases: []string{"stacks"},
	}

	cmdutils.EnableRepoOverride(stackCmd, f)

	stackCmd.AddCommand(stackCreateCmd.NewCmdCreateStack(f))
	stackCmd.AddCommand(stackSaveCmd.NewCmdSaveStack(f))
	stackCmd.AddCommand(stackSaveCmd.NewCmdAmendStack(f))
	stackCmd.AddCommand(stackSyncCmd.NewCmdSyncStack(f))
	stackCmd.AddCommand(stackMoveCmd.NewCmdStackPrev(f))
	stackCmd.AddCommand(stackMoveCmd.NewCmdStackNext(f))
	stackCmd.AddCommand(stackMoveCmd.NewCmdStackFirst(f))
	stackCmd.AddCommand(stackMoveCmd.NewCmdStackLast(f))
	stackCmd.AddCommand(stackMoveCmd.NewCmdStackMove(f))

	return stackCmd
}
