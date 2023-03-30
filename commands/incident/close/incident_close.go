package close

import (
	"github.com/spf13/cobra"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/commands/issuable"

	issuableCloseCmd "gitlab.com/gitlab-org/cli/commands/issuable/close"
)

func NewCmdClose(f *cmdutils.Factory) *cobra.Command {
	return issuableCloseCmd.NewCmdClose(f, issuable.TypeIncident)
}
