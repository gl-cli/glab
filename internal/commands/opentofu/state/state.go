package state

import (
	"github.com/spf13/cobra"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	listCmd "gitlab.com/gitlab-org/cli/internal/commands/opentofu/state/list"
	lockCmd "gitlab.com/gitlab-org/cli/internal/commands/opentofu/state/lock"
)

func NewCmd(f cmdutils.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "state <command> [flags]",
		Short: `Work with the OpenTofu / Terraform states.`,
	}

	cmd.AddCommand(listCmd.NewCmd(f))
	cmd.AddCommand(lockCmd.NewCmd(f))
	return cmd
}
