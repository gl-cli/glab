package subscribe

import (
	"github.com/spf13/cobra"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/commands/issuable"

	issuableSubscribeCmd "gitlab.com/gitlab-org/cli/commands/issuable/subscribe"
)

func NewCmdSubscribe(f *cmdutils.Factory) *cobra.Command {
	return issuableSubscribeCmd.NewCmdSubscribe(f, issuable.TypeIncident)
}
