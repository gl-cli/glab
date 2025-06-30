package list

import (
	"github.com/spf13/cobra"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/commands/issuable"
	issuableListCmd "gitlab.com/gitlab-org/cli/internal/commands/issuable/list"
)

func NewCmdList(f cmdutils.Factory, runE func(opts *issuableListCmd.ListOptions) error) *cobra.Command {
	return issuableListCmd.NewCmdList(f, runE, issuable.TypeIncident)
}
