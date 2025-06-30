package list

import (
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/git"
	"gitlab.com/gitlab-org/cli/internal/iostreams"
	"gitlab.com/gitlab-org/cli/internal/text"
)

func NewCmdStackList(f cmdutils.Factory, gr git.GitRunner) *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "Lists all entries in the stack. (EXPERIMENTAL.)",
		Long:    "Lists all entries in the stack. To select a different revision, use a command like 'stack move'.\n" + text.ExperimentalString,
		Example: heredoc.Doc(`
			$ glab stack list
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			title, err := git.GetCurrentStackTitle()
			if err != nil {
				return err
			}

			stack, err := git.GatherStackRefs(title)
			if err != nil {
				return err
			}

			currentBranch, err := git.CurrentBranch()
			if err != nil {
				return err
			}

			run(f.IO(), stack, currentBranch)
			return nil
		},
	}
}

func run(io *iostreams.IOStreams, stack git.Stack, currentBranch string) {
	c := io.Color()
	for ref := range stack.Iter() {
		if currentBranch == ref.Branch {
			fmt.Fprintf(io.StdOut, "> %s", c.Bold(ref.Branch))
		} else {
			fmt.Fprintf(io.StdOut, "  %s", ref.Branch)
		}
		fmt.Fprintf(io.StdOut, " - %s\n", c.Cyan(ref.Subject()))
	}
}
