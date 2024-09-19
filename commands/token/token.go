package token

import (
	"github.com/spf13/cobra"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/commands/token/create"
)

func NewTokenCmd(f *cmdutils.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "token",
		Short:   "Manage personal, project, or group tokens",
		Aliases: []string{"tok"},
	}

	cmdutils.EnableRepoOverride(cmd, f)
	cmd.AddCommand(create.NewCmdCreate(f, nil))
	return cmd
}
