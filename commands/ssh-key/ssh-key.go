package ssh

import (
	"github.com/spf13/cobra"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	cmdAdd "gitlab.com/gitlab-org/cli/commands/ssh-key/add"
	cmdDelete "gitlab.com/gitlab-org/cli/commands/ssh-key/delete"
	cmdGet "gitlab.com/gitlab-org/cli/commands/ssh-key/get"
	cmdList "gitlab.com/gitlab-org/cli/commands/ssh-key/list"
)

func NewCmdSSHKey(f *cmdutils.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ssh-key <command>",
		Short: "Manage SSH keys registered with your GitLab account.",
		Long:  "",
	}

	cmdutils.EnableRepoOverride(cmd, f)

	cmd.AddCommand(cmdAdd.NewCmdAdd(f, nil))
	cmd.AddCommand(cmdGet.NewCmdGet(f, nil))
	cmd.AddCommand(cmdList.NewCmdList(f, nil))
	cmd.AddCommand(cmdDelete.NewCmdDelete(f, nil))

	return cmd
}
