package create

import (
	"fmt"

	"gitlab.com/gitlab-org/cli/pkg/prompt"

	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
)

var boardName string

func NewCmdCreate(f *cmdutils.Factory) *cobra.Command {
	issueCmd := &cobra.Command{
		Use:     "create [flags]",
		Short:   `Create a project issue board.`,
		Long:    ``,
		Aliases: []string{"new"},
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 {
				boardName = args[0]
			}
			var err error
			out := f.IO.StdOut
			c := f.IO.Color()

			apiClient, err := f.HttpClient()
			if err != nil {
				return err
			}

			repo, err := f.BaseRepo()
			if err != nil {
				return err
			}

			if boardName == "" {
				err = prompt.AskQuestionWithInput(&boardName, "board name", "Board Name:", "", true)
				if err != nil {
					return err
				}
			}

			opts := &gitlab.CreateIssueBoardOptions{
				Name: gitlab.Ptr(boardName),
			}

			fmt.Fprintln(out, "- Creating board")

			issueBoard, err := api.CreateIssueBoard(apiClient, repo.FullName(), opts)
			if err != nil {
				return err
			}

			fmt.Fprintf(out, "%s Board created: %q", c.GreenCheck(), issueBoard.Name)

			return nil
		},
	}

	issueCmd.Flags().StringVarP(&boardName, "name", "n", "", "The name of the new board")

	return issueCmd
}
