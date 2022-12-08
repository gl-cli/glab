package delete

import (
	"fmt"
	"strconv"
	"strings"

	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/pkg/utils"

	"github.com/MakeNowJust/heredoc"
	"github.com/spf13/cobra"
	"github.com/xanzy/go-gitlab"
)

func NewCmdDelete(f *cmdutils.Factory) *cobra.Command {
	pipelineDeleteCmd := &cobra.Command{
		Use:   "delete <id> [flags]",
		Short: `Delete a CI/CD pipeline`,
		Example: heredoc.Doc(`
	glab ci delete 34
	glab ci delete 12,34,2
	`),
		Long: ``,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			c := f.IO.Color()
			apiClient, err := f.HttpClient()
			if err != nil {
				return err
			}

			repo, err := f.BaseRepo()
			if err != nil {
				return err
			}

			if m, _ := cmd.Flags().GetString("status"); m != "" {
				l := &gitlab.ListProjectPipelinesOptions{}
				l.Status = gitlab.BuildState(gitlab.BuildStateValue(m))
				pipes, err := api.ListProjectPipelines(apiClient, repo.FullName(), l)
				if err != nil {
					return err
				}
				for _, item := range pipes {
					err := api.DeletePipeline(apiClient, repo.FullName(), item.ID)
					if err != nil {
						return err
					}

					fmt.Fprintln(f.IO.StdOut, c.RedCheck(), "Pipeline #"+strconv.Itoa(item.ID)+" deleted successfully")
				}

			} else {
				pipelineID := args[0]

				arrIds := strings.Split(strings.Trim(pipelineID, "[] "), ",")
				for _, i2 := range arrIds {
					fmt.Fprintln(f.IO.StdOut, "Deleting pipeline #"+i2)
					err := api.DeletePipeline(apiClient, repo.FullName(), utils.StringToInt(i2))
					if err != nil {
						return err
					}

					fmt.Fprintln(f.IO.StdOut, c.RedCheck(), "Pipeline #"+i2+" deleted successfully")
				}
				fmt.Println()
			}

			return nil
		},
	}

	pipelineDeleteCmd.Flags().StringP("status", "s", "", "delete pipelines by status: {running|pending|success|failed|canceled|skipped|created|manual}")

	return pipelineDeleteCmd
}
