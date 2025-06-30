package create

import (
	"fmt"
	"strings"
	"time"

	"github.com/MakeNowJust/heredoc/v2"
	"gitlab.com/gitlab-org/cli/internal/git"
	"gitlab.com/gitlab-org/cli/internal/prompt"
	"gitlab.com/gitlab-org/cli/internal/text"
	"gitlab.com/gitlab-org/cli/internal/utils"

	"github.com/briandowns/spinner"
	"github.com/spf13/cobra"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
)

var longString = `Create a new stacked diff. Adds metadata to your "./.git/stacked" directory.
` + text.ExperimentalString

func NewCmdCreateStack(f cmdutils.Factory, gr git.GitRunner) *cobra.Command {
	stackCreateCmd := &cobra.Command{
		Use:     "create",
		Short:   "Create a new stacked diff. (EXPERIMENTAL.)",
		Long:    strings.Replace(longString, "\"./.git/stacked\"", "`./.git/stacked`", 1),
		Aliases: []string{"new"},
		Example: heredoc.Doc(`
			$ glab stack create cool-new-feature
			$ glab stack new cool-new-feature
		`),
		Args: cobra.MaximumNArgs(10),
		RunE: func(cmd *cobra.Command, args []string) error {
			var titleString string

			if len(args) == 1 {
				titleString = args[0]
			} else if len(args) == 0 {
				err := prompt.AskQuestionWithInput(&titleString, "title", "New stack title?", "", true)
				if err != nil {
					return fmt.Errorf("error prompting for title: %v", err)
				}
			} else {
				titleString = strings.Join(args[:], "-")
			}

			s := spinner.New(spinner.CharSets[11], 100*time.Millisecond)
			color := f.IO().Color()

			title := utils.ReplaceNonAlphaNumericChars(titleString, "-")
			if title != titleString {
				fmt.Fprintf(f.IO().StdErr, "%s warning: invalid characters have been replaced with dashes: %s\n",
					color.WarnIcon(),
					color.Blue(title))
			}

			err := git.SetLocalConfig("glab.currentstack", title)
			if err != nil {
				return fmt.Errorf("error setting local Git config: %v", err)
			}

			_, err = git.AddStackRefDir(title)
			if err != nil {
				return fmt.Errorf("error adding stack metadata directory: %v", err)
			}

			currentBranch, err := gr.Git("symbolic-ref", "--quiet", "--short", "HEAD")
			if err != nil {
				return fmt.Errorf("error getting current branch: %w", err)
			}

			err = git.AddStackBaseBranch(title, currentBranch)
			if err != nil {
				return fmt.Errorf("error adding current branch to metadata: %w", err)
			}

			if f.IO().IsOutputTTY() {
				fmt.Fprintf(f.IO().StdOut, "New stack created with title \"%s\".\n", title)
			}

			s.Stop()

			return nil
		},
	}
	return stackCreateCmd
}
