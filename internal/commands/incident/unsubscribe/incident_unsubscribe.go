package unsubscribe

import (
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/commands/issuable"

	issuableUnsubscribeCmd "gitlab.com/gitlab-org/cli/internal/commands/issuable/unsubscribe"

	"github.com/spf13/cobra"
)

func NewCmdUnsubscribe(f cmdutils.Factory) *cobra.Command {
	return issuableUnsubscribeCmd.NewCmdUnsubscribe(f, issuable.TypeIncident)
}
