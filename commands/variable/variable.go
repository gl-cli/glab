package variable

import (
	"github.com/spf13/cobra"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	deleteCmd "gitlab.com/gitlab-org/cli/commands/variable/delete"
	exportCmd "gitlab.com/gitlab-org/cli/commands/variable/export"
	getCmd "gitlab.com/gitlab-org/cli/commands/variable/get"
	listCmd "gitlab.com/gitlab-org/cli/commands/variable/list"
	setCmd "gitlab.com/gitlab-org/cli/commands/variable/set"
	updateCmd "gitlab.com/gitlab-org/cli/commands/variable/update"
)

func NewVariableCmd(f *cmdutils.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "variable",
		Short:   "Manage variables for a GitLab project or group.",
		Aliases: []string{"var"},
	}

	cmdutils.EnableRepoOverride(cmd, f)

	cmd.AddCommand(setCmd.NewCmdSet(f, nil))
	cmd.AddCommand(listCmd.NewCmdSet(f, nil))
	cmd.AddCommand(deleteCmd.NewCmdSet(f, nil))
	cmd.AddCommand(updateCmd.NewCmdSet(f, nil))
	cmd.AddCommand(getCmd.NewCmdSet(f, nil))
	cmd.AddCommand(exportCmd.NewCmdExport(f, nil))
	return cmd
}
