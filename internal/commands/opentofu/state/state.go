package state

import (
	"github.com/spf13/cobra"

	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	deleteCmd "gitlab.com/gitlab-org/cli/internal/commands/opentofu/state/delete"
	downloadCmd "gitlab.com/gitlab-org/cli/internal/commands/opentofu/state/download"
	listCmd "gitlab.com/gitlab-org/cli/internal/commands/opentofu/state/list"
	lockCmd "gitlab.com/gitlab-org/cli/internal/commands/opentofu/state/lock"
	unlockCmd "gitlab.com/gitlab-org/cli/internal/commands/opentofu/state/unlock"
)

func NewCmd(f cmdutils.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "state <command> [flags]",
		Short: `Work with the OpenTofu or Terraform states.`,
	}

	cmd.AddCommand(listCmd.NewCmd(f))
	cmd.AddCommand(lockCmd.NewCmd(f))
	cmd.AddCommand(unlockCmd.NewCmd(f))
	cmd.AddCommand(downloadCmd.NewCmd(f))
	cmd.AddCommand(deleteCmd.NewCmd(f))
	return cmd
}
