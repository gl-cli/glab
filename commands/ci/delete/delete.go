package delete

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/xanzy/go-gitlab"

	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"

	"github.com/MakeNowJust/heredoc"
	"github.com/spf13/cobra"
)

func NewCmdDelete(f *cmdutils.Factory) *cobra.Command {
	pipelineDeleteCmd := &cobra.Command{
		Use:   "delete <id> [flags]",
		Short: `Delete a CI/CD pipeline`,
		Example: heredoc.Doc(`
	glab ci delete 34
	glab ci delete 12,34,2
	glab ci delete --status=failed
	`),
		Long: ``,
		Args: func(cmd *cobra.Command, args []string) error {
			if m, _ := cmd.Flags().GetString("status"); m != "" && len(args) > 0 {
				return fmt.Errorf("either a status filter or a pipeline id must be passed, but not both")
			} else if m == "" {
				return cobra.ExactArgs(1)(cmd, args)
			} else {
				return nil
			}
		},
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

			var pipelineIDs []int

			if m, _ := cmd.Flags().GetString("status"); m != "" {
				pipes, err := api.ListProjectPipelines(apiClient, repo.FullName(), &gitlab.ListProjectPipelinesOptions{
					Status: gitlab.BuildState(gitlab.BuildStateValue(m)),
				})
				if err != nil {
					return err
				}

				for _, item := range pipes {
					pipelineIDs = append(pipelineIDs, item.ID)
				}
			} else {
				for _, stringID := range strings.Split(strings.Trim(args[0], "[] "), ",") {
					id, err := strconv.Atoi(stringID)
					if err != nil {
						return err
					}
					pipelineIDs = append(pipelineIDs, id)
				}
			}

			for _, id := range pipelineIDs {
				if dryRun, _ := cmd.Flags().GetBool("dry-run"); dryRun {
					fmt.Fprintf(f.IO.StdOut, "%s Pipeline #%d will be deleted\n", c.DotWarnIcon(), id)
				} else {
					err := api.DeletePipeline(apiClient, repo.FullName(), id)
					if err != nil {
						return err
					}

					fmt.Fprintf(f.IO.StdOut, "%s Pipeline #%d deleted successfully\n", c.RedCheck(), id)
				}
			}
			fmt.Println()

			return nil
		},
	}

	pipelineDeleteCmd.Flags().BoolP("dry-run", "", false, "simulate process, but do not delete anything")
	pipelineDeleteCmd.Flags().StringP("status", "s", "", "delete pipelines by status: {running|pending|success|failed|canceled|skipped|created|manual}")

	return pipelineDeleteCmd
}
