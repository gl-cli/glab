package save

import (
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/briandowns/spinner"
	"gitlab.com/gitlab-org/cli/internal/run"
	"gitlab.com/gitlab-org/cli/pkg/git"
	"gitlab.com/gitlab-org/cli/pkg/text"
	"golang.org/x/crypto/sha3"

	"github.com/spf13/cobra"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
)

var description string

func NewCmdSaveStack(f *cmdutils.Factory, getText cmdutils.GetTextUsingEditor) *cobra.Command {
	stackSaveCmd := &cobra.Command{
		Use:   "save",
		Short: `Save your progress within a stacked diff. (EXPERIMENTAL.)`,
		Long: `Save your current progress with a diff on the stack.
` + text.ExperimentalString,
		Example: heredoc.Doc(`
			glab stack save added_file
			glab stack save . -m "added a function"
			glab stack save -m "added a function"`),
		RunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Flags().Changed("message") && cmd.Flags().Changed("description") {
				return &cmdutils.FlagError{Err: errors.New("specify either of --message or --description.")}
			}

			// check if there are even any changes before we start
			err := checkForChanges()
			if err != nil {
				return fmt.Errorf("could not save: %v", err)
			}

			// a description is required, so ask if one is not provided
			if description == "" {
				description, err = promptForCommit(f, getText, "")
				if err != nil {
					return fmt.Errorf("error getting commit message: %v", err)
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
			sha, err := generateStackSha(description, title, string(author), time.Now())
			if err != nil {
				return fmt.Errorf("error generating hash for stack branch name: %v", err)
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
			if !stack.Empty() {
				lastRef := stack.Last()

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
	stackSaveCmd.Flags().StringVarP(&description, "description", "d", "", "Description of the change.")
	stackSaveCmd.Flags().StringVarP(&description, "message", "m", "", "Alias for the description flag.")

	return stackSaveCmd
}

func checkForChanges() error {
	gitCmd := git.GitCommand("status", "--porcelain")
	output, err := run.PrepareCmd(gitCmd).Output()
	if err != nil {
		return fmt.Errorf("error running Git status: %v", err)
	}

	if string(output) == "" {
		return fmt.Errorf("no changes to save.")
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

func generateStackSha(message string, title string, author string, timestamp time.Time) (string, error) {
	toSha := []byte(message + title + author + timestamp.String())
	hashData := make([]byte, 4)

	shakeHash := sha3.NewShake256()
	shakeHash.Write(toSha)
	_, err := shakeHash.Read(hashData)
	if err != nil {
		return "", fmt.Errorf("error generating hash for stack branch: %v", err)
	}

	return hex.EncodeToString(hashData), nil
}

func createShaBranch(f *cmdutils.Factory, sha string, title string) (string, error) {
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

	branchTitle := []string{prefix, title, sha}

	branch := strings.Join(branchTitle, "-")
	return string(branch), nil
}
