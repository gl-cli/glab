package stackswitch

import (
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"

	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/pkg/git"
	"gitlab.com/gitlab-org/cli/pkg/text"
)

func NewCmdStackSwitch(f *cmdutils.Factory) *cobra.Command {
	stackSwitchCmd := &cobra.Command{
		Use:   "switch <stack-name>",
		Short: "Switch between stacks. (EXPERIMENTAL.)",
		Long: heredoc.Doc(
			"Switch between stacks to work on another stack created with \"glab stack create\".\n" +
				"To see the list of all stacks, check the `.git/stacked/` directory.\n" +
				text.ExperimentalString,
		),
		Example: "glab stack switch <stack-name>",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := switchFunc(f, args[0]); err != nil {
				return fmt.Errorf("switching stacks failed: %w", err)
			}
			return nil
		},
		Args: cobra.ExactArgs(1),
	}
	return stackSwitchCmd
}

func switchFunc(f *cmdutils.Factory, name string) error {
	currentStackTitle, err := git.GetCurrentStackTitle()
	if err != nil {
		return fmt.Errorf("error getting current stack: %v", err)
	}
	if currentStackTitle == name {
		// No need to switch, we're already on the right stack
		return nil
	}

	stacks, err := git.GetStacks()
	if err != nil {
		return fmt.Errorf("getting stacks: %v", err)
	}
	var foundStack *git.Stack
	for _, s := range stacks {
		if s.Title == name {
			foundStack = &s
			break
		}
	}
	if foundStack == nil {
		return fmt.Errorf("no stack named %q found", name)
	}

	err = git.SetLocalConfig("glab.currentstack", name)
	if err != nil {
		return fmt.Errorf("error setting local Git config: %w", err)
	}

	fmt.Fprintf(f.IO.StdOut, "Switched to stack %s.\n", name)
	return nil
}
