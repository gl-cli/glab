package unsubscribe

import (
	"github.com/spf13/cobra"

	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/commands/issuable"
	issuableUnsubscribeCmd "gitlab.com/gitlab-org/cli/internal/commands/issuable/unsubscribe"
)

func NewCmdUnsubscribe(f cmdutils.Factory) *cobra.Command {
	return issuableUnsubscribeCmd.NewCmdUnsubscribe(f, issuable.TypeIssue)
}
