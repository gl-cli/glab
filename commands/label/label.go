package label

import (
	"github.com/spf13/cobra"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	labelCreateCmd "gitlab.com/gitlab-org/cli/commands/label/create"
	labelListCmd "gitlab.com/gitlab-org/cli/commands/label/list"
)

func NewCmdLabel(f *cmdutils.Factory) *cobra.Command {
	labelCmd := &cobra.Command{
		Use:   "label <command> [flags]",
		Short: `Manage labels on remote.`,
		Long:  ``,
	}

	cmdutils.EnableRepoOverride(labelCmd, f)

	labelCmd.AddCommand(labelListCmd.NewCmdList(f))
	labelCmd.AddCommand(labelCreateCmd.NewCmdCreate(f))
	return labelCmd
}
