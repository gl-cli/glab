package opentofu

import (
	"github.com/spf13/cobra"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	initCmd "gitlab.com/gitlab-org/cli/internal/commands/opentofu/init"
)

func NewCmd(f cmdutils.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "opentofu <command> [flags]",
		Short: `Work with the OpenTofu / Terraform integration.`,
		Long:  ``,
	}

	cmdutils.EnableRepoOverride(cmd, f)

	cmd.AddCommand(initCmd.NewCmd(f, initCmd.RunCommand))
	return cmd
}
