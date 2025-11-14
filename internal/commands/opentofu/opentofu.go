package opentofu

import (
	"github.com/spf13/cobra"

	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	initCmd "gitlab.com/gitlab-org/cli/internal/commands/opentofu/init"
	stateCmd "gitlab.com/gitlab-org/cli/internal/commands/opentofu/state"
)

func NewCmd(f cmdutils.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "opentofu <command> [flags]",
		Short:   `Work with the OpenTofu or Terraform integration.`,
		Long:    ``,
		Aliases: []string{"terraform", "tf"},
	}

	cmdutils.EnableRepoOverride(cmd, f)

	cmd.AddCommand(initCmd.NewCmd(f, initCmd.RunCommand))
	cmd.AddCommand(stateCmd.NewCmd(f))
	return cmd
}
