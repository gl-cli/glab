package stack

import (
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	stackCreateCmd "gitlab.com/gitlab-org/cli/internal/commands/stack/create"
	stackListCmd "gitlab.com/gitlab-org/cli/internal/commands/stack/list"
	stackMoveCmd "gitlab.com/gitlab-org/cli/internal/commands/stack/navigate"
	stackReorderCmd "gitlab.com/gitlab-org/cli/internal/commands/stack/reorder"
	stackSaveCmd "gitlab.com/gitlab-org/cli/internal/commands/stack/save"
	stackSwitchCmd "gitlab.com/gitlab-org/cli/internal/commands/stack/switch"
	stackSyncCmd "gitlab.com/gitlab-org/cli/internal/commands/stack/sync"
	"gitlab.com/gitlab-org/cli/internal/git"
	"gitlab.com/gitlab-org/cli/internal/surveyext"
	"gitlab.com/gitlab-org/cli/internal/text"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
)

func wrappedEdit(f cmdutils.Factory) cmdutils.GetTextUsingEditor {
	return func(editor, tmpFileName, content string) (string, error) {
		return surveyext.Edit(editor, tmpFileName, content, f.IO().In, f.IO().StdOut, f.IO().StdErr, nil)
	}
}

func NewCmdStack(f cmdutils.Factory) *cobra.Command {
	stackCmd := &cobra.Command{
		Use:   "stack <command> [flags]",
		Short: `Create, manage, and work with stacked diffs. (EXPERIMENTAL.)`,
		Long:  `Stacked diffs are a way of creating small changes that build upon each other to ultimately deliver a feature. This kind of workflow can be used to accelerate development time by continuing to build upon your changes, while earlier changes in the stack are reviewed and updated based on feedback.` + "\n" + text.ExperimentalString,
		Example: heredoc.Doc(`
			$ glab stack create cool-new-feature
			$ glab stack sync
		`),
		Aliases: []string{"stacks"},
	}

	var gr git.StandardGitCommand

	cmdutils.EnableRepoOverride(stackCmd, f)
	getTextFromEditor := wrappedEdit(f)

	stackCmd.AddCommand(stackCreateCmd.NewCmdCreateStack(f, gr))
	stackCmd.AddCommand(stackSaveCmd.NewCmdSaveStack(f, gr, getTextFromEditor))
	stackCmd.AddCommand(stackSaveCmd.NewCmdAmendStack(f, gr, getTextFromEditor))
	stackCmd.AddCommand(stackSyncCmd.NewCmdSyncStack(f, gr))
	stackCmd.AddCommand(stackMoveCmd.NewCmdStackPrev(f, gr))
	stackCmd.AddCommand(stackMoveCmd.NewCmdStackNext(f, gr))
	stackCmd.AddCommand(stackMoveCmd.NewCmdStackFirst(f, gr))
	stackCmd.AddCommand(stackMoveCmd.NewCmdStackLast(f, gr))
	stackCmd.AddCommand(stackMoveCmd.NewCmdStackMove(f, gr))
	stackCmd.AddCommand(stackListCmd.NewCmdStackList(f, gr))
	stackCmd.AddCommand(stackReorderCmd.NewCmdReorderStack(f, gr, getTextFromEditor))
	stackCmd.AddCommand(stackSwitchCmd.NewCmdStackSwitch(f, gr))

	return stackCmd
}
