package delete

import (
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"

	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
)

func NewCmdDelete(f cmdutils.Factory) *cobra.Command {
	labelDeleteCmd := &cobra.Command{
		Use:   "delete [flags]",
		Short: `Delete labels for a repository or project.`,
		Long:  ``,
		Example: heredoc.Doc(`
			$ glab label delete foo
			$ glab label delete -R owner/repo foo
		`),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error

			apiClient, err := f.HttpClient()
			if err != nil {
				return err
			}

			repo, err := f.BaseRepo()
			if err != nil {
				return err
			}

			o := &gitlab.DeleteLabelOptions{}

			_, err = apiClient.Labels.DeleteLabel(repo.FullName(), args[0], o)
			if err != nil {
				return err
			}
			fmt.Fprintf(f.IO().StdOut, "Label deleted")

			return nil
		},
	}

	return labelDeleteCmd
}
