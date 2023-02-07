package view

import (
	"github.com/spf13/cobra"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/commands/issuable"
	issuableViewCmd "gitlab.com/gitlab-org/cli/commands/issuable/view"
)

func NewCmdView(f *cmdutils.Factory) *cobra.Command {
	return issuableViewCmd.NewCmdView(f, issuable.TypeIncident)
}
