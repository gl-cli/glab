package auth

import (
	"github.com/spf13/cobra"
	authLoginCmd "gitlab.com/gitlab-org/cli/commands/auth/login"
	authStatusCmd "gitlab.com/gitlab-org/cli/commands/auth/status"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
)

func NewCmdAuth(f *cmdutils.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth <command>",
		Short: "Manage glab's authentication state.",
	}

	cmd.AddCommand(authLoginCmd.NewCmdLogin(f))
	cmd.AddCommand(authStatusCmd.NewCmdStatus(f, nil))
	cmd.AddCommand(authLoginCmd.NewCmdCredential(f, nil))

	return cmd
}
