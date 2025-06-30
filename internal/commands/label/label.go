package label

import (
	"github.com/spf13/cobra"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	labelCreateCmd "gitlab.com/gitlab-org/cli/internal/commands/label/create"
	labelDeleteCmd "gitlab.com/gitlab-org/cli/internal/commands/label/delete"
	labelListCmd "gitlab.com/gitlab-org/cli/internal/commands/label/list"
)

func NewCmdLabel(f cmdutils.Factory) *cobra.Command {
	labelCmd := &cobra.Command{
		Use:   "label <command> [flags]",
		Short: `Manage labels on remote.`,
		Long:  ``,
	}

	cmdutils.EnableRepoOverride(labelCmd, f)

	labelCmd.AddCommand(labelListCmd.NewCmdList(f))
	labelCmd.AddCommand(labelCreateCmd.NewCmdCreate(f))
	labelCmd.AddCommand(labelDeleteCmd.NewCmdDelete(f))

	return labelCmd
}
