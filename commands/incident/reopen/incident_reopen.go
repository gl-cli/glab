package reopen

import (
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/commands/issuable"

	"github.com/spf13/cobra"

	issuableReopenCmd "gitlab.com/gitlab-org/cli/commands/issuable/reopen"
)

func NewCmdReopen(f *cmdutils.Factory) *cobra.Command {
	return issuableReopenCmd.NewCmdReopen(f, issuable.TypeIncident)
}
