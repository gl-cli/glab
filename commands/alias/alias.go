package alias

import (
	deleteCmd "gitlab.com/gitlab-org/cli/commands/alias/delete"
	listCmd "gitlab.com/gitlab-org/cli/commands/alias/list"
	setCmd "gitlab.com/gitlab-org/cli/commands/alias/set"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"

	"github.com/spf13/cobra"
)

func NewCmdAlias(f *cmdutils.Factory) *cobra.Command {
	aliasCmd := &cobra.Command{
		Use:   "alias [command] [flags]",
		Short: `Create, list, and delete aliases.`,
		Long:  ``,
	}
	aliasCmd.AddCommand(deleteCmd.NewCmdDelete(f, nil))
	aliasCmd.AddCommand(listCmd.NewCmdList(f, nil))
	aliasCmd.AddCommand(setCmd.NewCmdSet(f, nil))
	return aliasCmd
}
