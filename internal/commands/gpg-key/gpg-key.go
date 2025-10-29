package gpg

import (
	"github.com/spf13/cobra"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	cmdAdd "gitlab.com/gitlab-org/cli/internal/commands/gpg-key/add"
	cmdDelete "gitlab.com/gitlab-org/cli/internal/commands/gpg-key/delete"
	cmdGet "gitlab.com/gitlab-org/cli/internal/commands/gpg-key/get"
	cmdList "gitlab.com/gitlab-org/cli/internal/commands/gpg-key/list"
)

func NewCmdGPGKey(f cmdutils.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gpg-key <command>",
		Short: "Manage GPG keys registered with your GitLab account.",
		Long:  "",
	}

	cmdutils.EnableRepoOverride(cmd, f)

	cmd.AddCommand(cmdAdd.NewCmdAdd(f))
	cmd.AddCommand(cmdGet.NewCmdGet(f))
	cmd.AddCommand(cmdList.NewCmdList(f))
	cmd.AddCommand(cmdDelete.NewCmdDelete(f))

	return cmd
}
