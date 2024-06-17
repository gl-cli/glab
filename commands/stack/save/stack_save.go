package save

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/MakeNowJust/heredoc"
	"github.com/briandowns/spinner"
	"gitlab.com/gitlab-org/cli/internal/run"
	"gitlab.com/gitlab-org/cli/pkg/git"
	"gitlab.com/gitlab-org/cli/pkg/prompt"
	"gitlab.com/gitlab-org/cli/pkg/text"

	"github.com/spf13/cobra"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
)

var description string

func NewCmdSaveStack(f *cmdutils.Factory) *cobra.Command {
	stackSaveCmd := &cobra.Command{
		Use:   "save",
		Short: `Save your progress within a stacked diff.`,
		Long: `Save your current progress with a diff on the stack.
` + text.ExperimentalString,
		Example: heredoc.Doc(`
			glab stack save added_file
			glab stack save . -m "added a function"
			glab stack save -m "added a function"`),
		RunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Flags().Changed("message") && cmd.Flags().Changed("description") {
				return &cmdutils.FlagError{Err: errors.New("specify either of --message or --description")}
			}

			// check if there are even any changes before we start
			err := checkForChanges()
			if err != nil {
				return fmt.Errorf("could not save: %v", err)
			}

			// a description is required, so ask if one is not provided
			if description == "" {
				err := prompt.AskQuestionWithInput(&description, "description", "How would you describe this change?", "", true)
				if err != nil {
					return fmt.Errorf("error prompting for save description: %v", err)
				}
			}

			s := spinner.New(spinner.CharSets[11], 100*time.Millisecond)

			// git add files
			_, err = addFiles(args[0:])
			if err != nil {
				return fmt.Errorf("error adding files: %v", err)
			}

			// get stack title
			title, err := git.GetCurrentStackTitle()
			if err != nil {
				return fmt.Errorf("error running Git command: %v", err)
			}

			author, err := git.GitUserName()
			if err != nil {
				return fmt.Errorf("error getting Git author: %v", err)
			}

			// generate a SHA based on: commit message, stack title, Git author name
			sha, err := generateStackSha(description, title, string(author))
			if err != nil {
				return fmt.Errorf("error generating SHA command: %v", err)
			}

			// create branch name from SHA
			branch, err := createShaBranch(f, sha, title)
			if err != nil {
				return fmt.Errorf("error creating branch name: %v", err)
			}

			// create the branch prefix-stack_title-SHA
			err = git.CheckoutNewBranch(branch)
			if err != nil {
				return fmt.Errorf("error running branch checkout: %v", err)
			}

			// commit files to branch
			_, err = commitFiles(description)
			if err != nil {
				return fmt.Errorf("error committing files: %v", err)
			}

			stack, err := git.GatherStackRefs(title)
			if err != nil {
				return fmt.Errorf("error getting refs from file system: %v", err)
			}

			var stackRef git.StackRef
			if len(stack.Refs) > 0 {
				lastRef, err := stack.Last()
				if err != nil {
					return fmt.Errorf("error getting last ref: %v", err)
				}

				// update the ref before it (the current last ref)
				err = git.UpdateStackRefFile(title, git.StackRef{
					Prev:        lastRef.Prev,
					MR:          lastRef.MR,
					Description: lastRef.Description,
					SHA:         lastRef.SHA,
					Branch:      lastRef.Branch,
					Next:        sha,
				})
				if err != nil {
					return fmt.Errorf("error updating old ref: %v", err)
				}

				stackRef = git.StackRef{Prev: lastRef.SHA, SHA: sha, Branch: branch, Description: description}
			} else {
				stackRef = git.StackRef{SHA: sha, Branch: branch, Description: description}
			}

			err = git.AddStackRefFile(title, stackRef)
			if err != nil {
				return fmt.Errorf("error creating stack file: %v", err)
			}

			if f.IO.IsOutputTTY() {
				color := f.IO.Color()

				fmt.Fprintf(
					f.IO.StdOut,
					"%s %s: Saved with message: \"%s\".\n",
					color.ProgressIcon(),
					color.Blue(title),
					description,
				)
			}

			s.Stop()

			return nil
		},
	}
	stackSaveCmd.Flags().StringVarP(&description, "description", "d", "", "a description of the change")
	stackSaveCmd.Flags().StringVarP(&description, "message", "m", "", "alias for the description flag")

	return stackSaveCmd
}

func checkForChanges() error {
	gitCmd := git.GitCommand("status", "--porcelain")
	output, err := run.PrepareCmd(gitCmd).Output()
	if err != nil {
		return fmt.Errorf("error running Git status: %v", err)
	}

	if string(output) == "" {
		return fmt.Errorf("no changes to save")
	}

	return nil
}

func addFiles(args []string) (files []string, err error) {
	if len(args) == 0 {
		args = []string{"."}
	}

	for _, file := range args {
		_, err = os.Stat(file)
		if err != nil {
			return
		}

		files = append(files, file)
	}

	cmdargs := append([]string{"add"}, args...)
	gitCmd := git.GitCommand(cmdargs...)

	_, err = run.PrepareCmd(gitCmd).Output()
	if err != nil {
		return []string{}, fmt.Errorf("error running Git add: %v", err)
	}

	return files, err
}

func commitFiles(message string) (string, error) {
	commitCmd := git.GitCommand("commit", "-m", message)
	output, err := run.PrepareCmd(commitCmd).Output()
	if err != nil {
		return "", fmt.Errorf("error running Git command: %v", err)
	}

	return string(output), nil
}

func generateStackSha(message string, title string, author string) (string, error) {
	toSha := message + title + author

	cmd := "echo \"" + toSha + "\" | git hash-object --stdin"
	sha, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		return "", fmt.Errorf("error running git hash-object: %v", err)
	}

	return strings.TrimSuffix(string(sha), "\n"), nil
}

func createShaBranch(f *cmdutils.Factory, sha string, title string) (string, error) {
	shortSha := string(sha)[0:8]

	cfg, err := f.Config()
	if err != nil {
		return "", fmt.Errorf("could not retrieve config file: %v", err)
	}

	prefix, err := cfg.Get("", "branch_prefix")
	if err != nil {
		return "", fmt.Errorf("could not get prefix config: %v", err)
	}

	if prefix == "" {
		prefix = os.Getenv("USER")
		if prefix == "" {
			prefix = "glab-stack"
		}
	}

	branchTitle := []string{prefix, title, shortSha}

	branch := strings.Join(branchTitle, "-")
	return string(branch), nil
}
