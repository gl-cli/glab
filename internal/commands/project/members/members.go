package members

import (
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	membersAdd "gitlab.com/gitlab-org/cli/internal/commands/project/members/add"
	membersRemove "gitlab.com/gitlab-org/cli/internal/commands/project/members/remove"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
)

func NewCmdMembers(f cmdutils.Factory) *cobra.Command {
	membersCmd := &cobra.Command{
		Use:   "members <command> [flags]",
		Short: `Manage project members.`,
		Long: heredoc.Doc(`
			Add or remove members from a GitLab project.
		`),
	}

	membersCmd.AddCommand(membersAdd.NewCmd(f))
	membersCmd.AddCommand(membersRemove.NewCmd(f))

	return membersCmd
}
