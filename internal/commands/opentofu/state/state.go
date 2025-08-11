package state

import (
	"github.com/spf13/cobra"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	listCmd "gitlab.com/gitlab-org/cli/internal/commands/opentofu/state/list"
)

func NewCmd(f cmdutils.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "state <command> [flags]",
		Short: `Work with the OpenTofu / Terraform states.`,
	}

	cmd.AddCommand(listCmd.NewCmd(f))
	return cmd
}
