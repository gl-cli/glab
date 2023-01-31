package view

import (
	"github.com/spf13/cobra"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/commands/issuable"
	issueViewCmd "gitlab.com/gitlab-org/cli/commands/issue/view"
)

func NewCmdView(f *cmdutils.Factory) *cobra.Command {
	return issueViewCmd.NewCmdViewByType(f, issuable.TypeIncident)
}
