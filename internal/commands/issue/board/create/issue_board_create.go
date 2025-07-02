package create

import (
	"fmt"

	"gitlab.com/gitlab-org/cli/internal/prompt"

	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
)

var createIssueBoard = func(client *gitlab.Client, projectID any, opts *gitlab.CreateIssueBoardOptions) (*gitlab.IssueBoard, error) {
	board, _, err := client.Boards.CreateIssueBoard(projectID, opts)
	if err != nil {
		return nil, err
	}

	return board, nil
}

func NewCmdCreate(f cmdutils.Factory) *cobra.Command {
	var boardName string
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
			out := f.IO().StdOut
			c := f.IO().Color()

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

			issueBoard, err := createIssueBoard(apiClient, repo.FullName(), opts)
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
