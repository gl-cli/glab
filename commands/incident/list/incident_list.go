package list

import (
	"github.com/spf13/cobra"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	issueListCmd "gitlab.com/gitlab-org/cli/commands/issue/list"
)

func NewCmdList(f *cmdutils.Factory, runE func(opts *issueListCmd.ListOptions) error) *cobra.Command {
	return issueListCmd.NewCmdListByType(f, runE, "incident")
}