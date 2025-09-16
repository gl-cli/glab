package token

import (
	"github.com/spf13/cobra"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"

	tokenListCmd "gitlab.com/gitlab-org/cli/internal/commands/cluster/agent/token/list"
	tokenRevokeCmd "gitlab.com/gitlab-org/cli/internal/commands/cluster/agent/token/revoke"
)

func NewCmd(f cmdutils.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "token <command> [flags]",
		Short: `Manage GitLab Agents for Kubernetes tokens.`,
		Long:  ``,
	}

	cmd.AddCommand(tokenListCmd.NewCmd(f))
	cmd.AddCommand(tokenRevokeCmd.NewCmd(f))
	return cmd
}
