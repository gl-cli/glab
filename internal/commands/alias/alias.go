package alias

import (
	"github.com/spf13/cobra"

	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	deleteCmd "gitlab.com/gitlab-org/cli/internal/commands/alias/delete"
	listCmd "gitlab.com/gitlab-org/cli/internal/commands/alias/list"
	setCmd "gitlab.com/gitlab-org/cli/internal/commands/alias/set"
)

func NewCmdAlias(f cmdutils.Factory) *cobra.Command {
	aliasCmd := &cobra.Command{
		Use:   "alias [command] [flags]",
		Short: `Create, list, and delete aliases.`,
		Long:  ``,
	}
	aliasCmd.AddCommand(deleteCmd.NewCmdDelete(f))
	aliasCmd.AddCommand(listCmd.NewCmdList(f))
	aliasCmd.AddCommand(setCmd.NewCmdSet(f))
	return aliasCmd
}
