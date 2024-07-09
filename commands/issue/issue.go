package issue

import (
	"github.com/MakeNowJust/heredoc/v2"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	issueBoardCmd "gitlab.com/gitlab-org/cli/commands/issue/board"
	issueCloseCmd "gitlab.com/gitlab-org/cli/commands/issue/close"
	issueCreateCmd "gitlab.com/gitlab-org/cli/commands/issue/create"
	issueDeleteCmd "gitlab.com/gitlab-org/cli/commands/issue/delete"
	issueListCmd "gitlab.com/gitlab-org/cli/commands/issue/list"
	issueNoteCmd "gitlab.com/gitlab-org/cli/commands/issue/note"
	issueReopenCmd "gitlab.com/gitlab-org/cli/commands/issue/reopen"
	issueSubscribeCmd "gitlab.com/gitlab-org/cli/commands/issue/subscribe"
	issueUnsubscribeCmd "gitlab.com/gitlab-org/cli/commands/issue/unsubscribe"
	issueUpdateCmd "gitlab.com/gitlab-org/cli/commands/issue/update"
	issueViewCmd "gitlab.com/gitlab-org/cli/commands/issue/view"

	"github.com/spf13/cobra"
)

func NewCmdIssue(f *cmdutils.Factory) *cobra.Command {
	issueCmd := &cobra.Command{
		Use:   "issue [command] [flags]",
		Short: `Work with GitLab issues.`,
		Long:  ``,
		Example: heredoc.Doc(`
			glab issue list
			glab issue create --label --confidential
			glab issue view --web 123
			glab issue note -m "closing because !123 was merged" <issue number>
		`),
		Annotations: map[string]string{
			"help:arguments": heredoc.Doc(`
				An issue can be supplied as argument in any of the following formats:
				- by number, like "123"
				- by URL, like "https://gitlab.com/NAMESPACE/REPO/-/issues/123"
			`),
		},
	}

	cmdutils.EnableRepoOverride(issueCmd, f)

	issueCmd.AddCommand(issueCloseCmd.NewCmdClose(f))
	issueCmd.AddCommand(issueBoardCmd.NewCmdBoard(f))
	issueCmd.AddCommand(issueCreateCmd.NewCmdCreate(f))
	issueCmd.AddCommand(issueDeleteCmd.NewCmdDelete(f))
	issueCmd.AddCommand(issueListCmd.NewCmdList(f, nil))
	issueCmd.AddCommand(issueNoteCmd.NewCmdNote(f))
	issueCmd.AddCommand(issueReopenCmd.NewCmdReopen(f))
	issueCmd.AddCommand(issueViewCmd.NewCmdView(f))
	issueCmd.AddCommand(issueSubscribeCmd.NewCmdSubscribe(f))
	issueCmd.AddCommand(issueUnsubscribeCmd.NewCmdUnsubscribe(f))
	issueCmd.AddCommand(issueUpdateCmd.NewCmdUpdate(f))
	return issueCmd
}
