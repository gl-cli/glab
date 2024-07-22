package todo

import (
	"fmt"

	"gitlab.com/gitlab-org/cli/commands/mr/mrutils"

	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"

	"github.com/spf13/cobra"
)

func NewCmdTodo(f *cmdutils.Factory) *cobra.Command {
	mrToDoCmd := &cobra.Command{
		Use:     "todo [<id> | <branch>]",
		Aliases: []string{"add-todo"},
		Short:   "Add a to-do item to merge request.",
		Long:    ``,
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			c := f.IO.Color()

			apiClient, err := f.HttpClient()
			if err != nil {
				return err
			}

			mr, repo, err := mrutils.MRFromArgs(f, args, "any")
			if err != nil {
				return err
			}

			_, err = api.MRTodo(apiClient, repo.FullName(), mr.IID, nil)
			if err != nil {
				return err
			}

			fmt.Fprintln(f.IO.StdOut, c.GreenCheck(), "Done!!")

			return nil
		},
	}

	return mrToDoCmd
}
