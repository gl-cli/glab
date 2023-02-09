package list

import (
	"github.com/spf13/cobra"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/commands/issuable"
	issuableListCmd "gitlab.com/gitlab-org/cli/commands/issuable/list"
)

func NewCmdList(f *cmdutils.Factory, runE func(opts *issuableListCmd.ListOptions) error) *cobra.Command {
	return issuableListCmd.NewCmdList(f, runE, issuable.TypeIssue)
}
