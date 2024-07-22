package save

import (
	"fmt"
	"time"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/briandowns/spinner"
	"gitlab.com/gitlab-org/cli/internal/run"
	"gitlab.com/gitlab-org/cli/pkg/git"
	"gitlab.com/gitlab-org/cli/pkg/prompt"
	"gitlab.com/gitlab-org/cli/pkg/text"

	"github.com/spf13/cobra"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
)

func NewCmdAmendStack(f *cmdutils.Factory) *cobra.Command {
	stackSaveCmd := &cobra.Command{
		Use:   "amend",
		Short: `Save more changes to a stacked diff. (EXPERIMENTAL.)`,
		Long: `Add more changes to an existing stacked diff.
` + text.ExperimentalString,
		Example: heredoc.Doc(`glab stack amend modifiedfile
			glab stack amend . -m "fixed a function"
			glab stack amend newfile -d "forgot to add this"`),
		RunE: func(cmd *cobra.Command, args []string) error {
			output, err := amendFunc(f, args, description)
			if err != nil {
				return fmt.Errorf("could not run stack amend: %v", err)
			}

			if f.IO.IsOutputTTY() {
				fmt.Fprint(f.IO.StdOut, output)
			}

			return nil
		},
	}
	stackSaveCmd.Flags().StringVarP(&description, "description", "d", "", "a description of the change")
	stackSaveCmd.Flags().StringVarP(&description, "message", "m", "", "alias for the description flag")
	stackSaveCmd.MarkFlagsMutuallyExclusive("message", "description")

	return stackSaveCmd
}

func amendFunc(f *cmdutils.Factory, args []string, description string) (string, error) {
	s := spinner.New(spinner.CharSets[11], 100*time.Millisecond)

	// check if there are even any changes before we start
	err := checkForChanges()
	if err != nil {
		return "", fmt.Errorf("could not save: %v", err)
	}

	// get stack title
	title, err := git.GetCurrentStackTitle()
	if err != nil {
		return "", fmt.Errorf("error running Git command: %v", err)
	}

	stack, err := checkForStack(title)
	if err != nil {
		return "", fmt.Errorf("error checking for stack: %v", err)
	}

	if stack.Branch == "" {
		return "", fmt.Errorf("not currently in a stack. Change to the branch you want to amend.")
	}

	// a description is required, so ask if one is not provided
	if description == "" {
		err := prompt.AskQuestionWithInput(&description, "description", "How would you describe this change?", stack.Description, true)
		if err != nil {
			return "", fmt.Errorf("error prompting for title description: %v", err)
		}
	}

	// git add files
	_, err = addFiles(args[0:])
	if err != nil {
		return "", fmt.Errorf("error adding files: %v", err)
	}

	// run the amend commit
	err = gitAmend(description)
	if err != nil {
		return "", fmt.Errorf("error amending commit with Git: %v", err)
	}

	var output string
	if f.IO.IsOutputTTY() {
		output = fmt.Sprintf("Amended stack item with description: %q.\n", description)
	}

	s.Stop()

	return output, nil
}

func gitAmend(description string) error {
	amendCmd := git.GitCommand("commit", "--amend", "-m", description)
	output, err := run.PrepareCmd(amendCmd).Output()
	if err != nil {
		return fmt.Errorf("error running Git command: %v", err)
	}

	fmt.Println("Amend commit: ", string(output))

	return nil
}

func checkForStack(title string) (git.StackRef, error) {
	stack, err := git.GatherStackRefs(title)
	if err != nil {
		return git.StackRef{}, err
	}

	branch, err := git.CurrentBranch()
	if err != nil {
		return git.StackRef{}, err
	}

	for _, ref := range stack.Refs {
		if ref.Branch == branch {
			return ref, nil
		}
	}

	return git.StackRef{}, nil
}
