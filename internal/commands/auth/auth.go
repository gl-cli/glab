package auth

import (
	"github.com/spf13/cobra"

	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	authDockerCredentialHelperCmd "gitlab.com/gitlab-org/cli/internal/commands/auth/docker"
	cmdGenerate "gitlab.com/gitlab-org/cli/internal/commands/auth/generate"
	authLoginCmd "gitlab.com/gitlab-org/cli/internal/commands/auth/login"
	authLogoutCmd "gitlab.com/gitlab-org/cli/internal/commands/auth/logout"
	authStatusCmd "gitlab.com/gitlab-org/cli/internal/commands/auth/status"
)

func NewCmdAuth(f cmdutils.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth <command>",
		Short: "Manage glab's authentication state.",
	}

	cmd.AddCommand(authLoginCmd.NewCmdLogin(f))
	cmd.AddCommand(authStatusCmd.NewCmdStatus(f, nil))
	cmd.AddCommand(authLoginCmd.NewCmdCredential(f))
	cmd.AddCommand(cmdGenerate.NewCmdGenerate(f))
	cmd.AddCommand(authLogoutCmd.NewCmdLogout(f))
	cmd.AddCommand(authDockerCredentialHelperCmd.NewCmdConfigureDocker(f))
	cmd.AddCommand(authDockerCredentialHelperCmd.NewCmdCredentialHelper(f))

	return cmd
}
