package unsubscribe

import (
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/commands/issuable"

	issuableUnsubscribeCmd "gitlab.com/gitlab-org/cli/commands/issuable/unsubscribe"

	"github.com/spf13/cobra"
)

func NewCmdUnsubscribe(f *cmdutils.Factory) *cobra.Command {
	return issuableUnsubscribeCmd.NewCmdUnsubscribe(f, issuable.TypeIssue)
}
