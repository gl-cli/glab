package close

import (
	"github.com/spf13/cobra"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/commands/issuable"

	issuableCloseCmd "gitlab.com/gitlab-org/cli/internal/commands/issuable/close"
)

func NewCmdClose(f cmdutils.Factory) *cobra.Command {
	return issuableCloseCmd.NewCmdClose(f, issuable.TypeIssue)
}
