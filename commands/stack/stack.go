package stack

import (
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	stackCreateCmd "gitlab.com/gitlab-org/cli/commands/stack/create"
	stackListCmd "gitlab.com/gitlab-org/cli/commands/stack/list"
	stackMoveCmd "gitlab.com/gitlab-org/cli/commands/stack/navigate"
	stackSaveCmd "gitlab.com/gitlab-org/cli/commands/stack/save"
	stackSwitchCmd "gitlab.com/gitlab-org/cli/commands/stack/switch"
	stackSyncCmd "gitlab.com/gitlab-org/cli/commands/stack/sync"
	"gitlab.com/gitlab-org/cli/pkg/surveyext"
	"gitlab.com/gitlab-org/cli/pkg/text"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
)

func wrappedEdit(f *cmdutils.Factory) cmdutils.GetTextUsingEditor {
	return func(editor, tmpFileName, content string) (string, error) {
		return surveyext.Edit(editor, tmpFileName, content, f.IO.In, f.IO.StdOut, f.IO.StdErr, nil)
	}
}

func NewCmdStack(f *cmdutils.Factory) *cobra.Command {
	stackCmd := &cobra.Command{
		Use:   "stack <command> [flags]",
		Short: `Create, manage, and work with stacked diffs. (EXPERIMENTAL.)`,
		Long:  `Stacked diffs are a way of creating small changes that build upon each other to ultimately deliver a feature. This kind of workflow can be used to accelerate development time by continuing to build upon your changes, while earlier changes in the stack are reviewed and updated based on feedback.` + "\n" + text.ExperimentalString,
		Example: heredoc.Doc(`
			glab stack create cool-new-feature
			glab stack sync
		`),
		Aliases: []string{"stacks"},
	}

	cmdutils.EnableRepoOverride(stackCmd, f)
	getTextFromEditor := wrappedEdit(f)

	stackCmd.AddCommand(stackCreateCmd.NewCmdCreateStack(f))
	stackCmd.AddCommand(stackSaveCmd.NewCmdSaveStack(f, getTextFromEditor))
	stackCmd.AddCommand(stackSaveCmd.NewCmdAmendStack(f, getTextFromEditor))
	stackCmd.AddCommand(stackSyncCmd.NewCmdSyncStack(f))
	stackCmd.AddCommand(stackMoveCmd.NewCmdStackPrev(f))
	stackCmd.AddCommand(stackMoveCmd.NewCmdStackNext(f))
	stackCmd.AddCommand(stackMoveCmd.NewCmdStackFirst(f))
	stackCmd.AddCommand(stackMoveCmd.NewCmdStackLast(f))
	stackCmd.AddCommand(stackMoveCmd.NewCmdStackMove(f))
	stackCmd.AddCommand(stackListCmd.NewCmdStackList(f))
	stackCmd.AddCommand(stackSwitchCmd.NewCmdStackSwitch(f))

	return stackCmd
}
