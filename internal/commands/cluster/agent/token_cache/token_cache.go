package token_cache

import (
	"github.com/spf13/cobra"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"

	"github.com/MakeNowJust/heredoc/v2"

	tokenCacheClearCmd "gitlab.com/gitlab-org/cli/internal/commands/cluster/agent/token_cache/clear"
	tokenCacheListCmd "gitlab.com/gitlab-org/cli/internal/commands/cluster/agent/token_cache/list"
)

func NewCmd(f cmdutils.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "token-cache <command> [flags]",
		Short: `Manage cached GitLab Agent tokens.`,
		Long: heredoc.Doc(`Manage cached GitLab Agent tokens created by 'glab cluster agent get-token'.

		This command group allows you to list and clear tokens that are cached locally
		in the keyring and filesystem cache.
		`),
	}

	cmd.AddCommand(tokenCacheListCmd.NewCmd(f))
	cmd.AddCommand(tokenCacheClearCmd.NewCmd(f))
	return cmd
}
