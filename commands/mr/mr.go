package mr

import (
	"github.com/MakeNowJust/heredoc/v2"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	mrApproveCmd "gitlab.com/gitlab-org/cli/commands/mr/approve"
	mrApproversCmd "gitlab.com/gitlab-org/cli/commands/mr/approvers"
	mrCheckoutCmd "gitlab.com/gitlab-org/cli/commands/mr/checkout"
	mrCloseCmd "gitlab.com/gitlab-org/cli/commands/mr/close"
	mrCreateCmd "gitlab.com/gitlab-org/cli/commands/mr/create"
	mrDeleteCmd "gitlab.com/gitlab-org/cli/commands/mr/delete"
	mrDiffCmd "gitlab.com/gitlab-org/cli/commands/mr/diff"
	mrForCmd "gitlab.com/gitlab-org/cli/commands/mr/for"
	mrIssuesCmd "gitlab.com/gitlab-org/cli/commands/mr/issues"
	mrListCmd "gitlab.com/gitlab-org/cli/commands/mr/list"
	mrMergeCmd "gitlab.com/gitlab-org/cli/commands/mr/merge"
	mrNoteCmd "gitlab.com/gitlab-org/cli/commands/mr/note"
	mrRebaseCmd "gitlab.com/gitlab-org/cli/commands/mr/rebase"
	mrReopenCmd "gitlab.com/gitlab-org/cli/commands/mr/reopen"
	mrRevokeCmd "gitlab.com/gitlab-org/cli/commands/mr/revoke"
	mrSubscribeCmd "gitlab.com/gitlab-org/cli/commands/mr/subscribe"
	mrTodoCmd "gitlab.com/gitlab-org/cli/commands/mr/todo"
	mrUnsubscribeCmd "gitlab.com/gitlab-org/cli/commands/mr/unsubscribe"
	mrUpdateCmd "gitlab.com/gitlab-org/cli/commands/mr/update"
	mrViewCmd "gitlab.com/gitlab-org/cli/commands/mr/view"

	"github.com/spf13/cobra"
)

func NewCmdMR(f *cmdutils.Factory) *cobra.Command {
	mrCmd := &cobra.Command{
		Use:   "mr <command> [flags]",
		Short: `Create, view, and manage merge requests.`,
		Long:  ``,
		Example: heredoc.Doc(`
			glab mr create --fill --label bugfix
			glab mr merge 123
			glab mr note -m "needs to do X before it can be merged" branch-foo
		`),
		Annotations: map[string]string{
			"help:arguments": heredoc.Doc(`
			A merge request can be supplied as argument in any of the following formats:
			- by number, e.g. "123"; or
			- by the name of its source branch, e.g. "patch-1" or "OWNER:patch-1".
			`),
		},
	}

	cmdutils.EnableRepoOverride(mrCmd, f)

	mrCmd.AddCommand(mrApproveCmd.NewCmdApprove(f))
	mrCmd.AddCommand(mrApproversCmd.NewCmdApprovers(f))
	mrCmd.AddCommand(mrCheckoutCmd.NewCmdCheckout(f))
	mrCmd.AddCommand(mrCloseCmd.NewCmdClose(f))
	mrCmd.AddCommand(mrCreateCmd.NewCmdCreate(f, nil))
	mrCmd.AddCommand(mrDeleteCmd.NewCmdDelete(f))
	mrCmd.AddCommand(mrDiffCmd.NewCmdDiff(f, nil))
	mrCmd.AddCommand(mrForCmd.NewCmdFor(f))
	mrCmd.AddCommand(mrIssuesCmd.NewCmdIssues(f))
	mrCmd.AddCommand(mrListCmd.NewCmdList(f, nil))
	mrCmd.AddCommand(mrMergeCmd.NewCmdMerge(f))
	mrCmd.AddCommand(mrNoteCmd.NewCmdNote(f))
	mrCmd.AddCommand(mrRebaseCmd.NewCmdRebase(f))
	mrCmd.AddCommand(mrReopenCmd.NewCmdReopen(f))
	mrCmd.AddCommand(mrRevokeCmd.NewCmdRevoke(f))
	mrCmd.AddCommand(mrSubscribeCmd.NewCmdSubscribe(f))
	mrCmd.AddCommand(mrUnsubscribeCmd.NewCmdUnsubscribe(f))
	mrCmd.AddCommand(mrTodoCmd.NewCmdTodo(f))
	mrCmd.AddCommand(mrUpdateCmd.NewCmdUpdate(f))
	mrCmd.AddCommand(mrViewCmd.NewCmdView(f))

	return mrCmd
}
