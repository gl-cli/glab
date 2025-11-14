package reopen

import (
	"github.com/spf13/cobra"

	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/commands/issuable"
	issuableReopenCmd "gitlab.com/gitlab-org/cli/internal/commands/issuable/reopen"
)

func NewCmdReopen(f cmdutils.Factory) *cobra.Command {
	return issuableReopenCmd.NewCmdReopen(f, issuable.TypeIssue)
}
