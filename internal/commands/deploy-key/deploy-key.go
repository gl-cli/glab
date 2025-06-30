package deploykey

import (
	"github.com/spf13/cobra"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	cmdAdd "gitlab.com/gitlab-org/cli/internal/commands/deploy-key/add"
	cmdDelete "gitlab.com/gitlab-org/cli/internal/commands/deploy-key/delete"
	cmdGet "gitlab.com/gitlab-org/cli/internal/commands/deploy-key/get"
	cmdList "gitlab.com/gitlab-org/cli/internal/commands/deploy-key/list"
)

func NewCmdDeployKey(f cmdutils.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deploy-key <command>",
		Short: "Manage deploy keys.",
		Long:  "",
	}

	cmdutils.EnableRepoOverride(cmd, f)

	cmd.AddCommand(cmdAdd.NewCmdAdd(f))
	cmd.AddCommand(cmdGet.NewCmdGet(f))
	cmd.AddCommand(cmdList.NewCmdList(f))
	cmd.AddCommand(cmdDelete.NewCmdDelete(f))

	return cmd
}
