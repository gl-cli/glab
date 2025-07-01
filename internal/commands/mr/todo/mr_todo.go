package todo

import (
	"errors"
	"fmt"
	"net/http"

	"gitlab.com/gitlab-org/cli/internal/commands/mr/mrutils"

	"gitlab.com/gitlab-org/cli/internal/cmdutils"

	"github.com/spf13/cobra"
)

var errTodoExists = errors.New("To-do already exists.")

func NewCmdTodo(f cmdutils.Factory) *cobra.Command {
	mrToDoCmd := &cobra.Command{
		Use:     "todo [<id> | <branch>]",
		Aliases: []string{"add-todo"},
		Short:   "Add a to-do item to merge request.",
		Long:    ``,
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			c := f.IO().Color()

			apiClient, err := f.HttpClient()
			if err != nil {
				return err
			}

			mr, repo, err := mrutils.MRFromArgs(f, args, "any")
			if err != nil {
				return err
			}

			_, resp, err := apiClient.MergeRequests.CreateTodo(repo.FullName(), mr.IID)

			if resp.StatusCode == http.StatusNotModified {
				return errTodoExists
			}
			if err != nil {
				return err
			}

			fmt.Fprintln(f.IO().StdOut, c.GreenCheck(), "Done!!")

			return nil
		},
	}

	return mrToDoCmd
}
