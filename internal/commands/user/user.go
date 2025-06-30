package user

import (
	"github.com/spf13/cobra"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	userEventsCmd "gitlab.com/gitlab-org/cli/internal/commands/user/events"
)

func NewCmdUser(f cmdutils.Factory) *cobra.Command {
	userCmd := &cobra.Command{
		Use:   "user <command> [flags]",
		Short: "Interact with a GitLab user account.",
		Long:  "",
	}

	userCmd.AddCommand(userEventsCmd.NewCmdEvents(f))

	return userCmd
}
