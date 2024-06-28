package list

import (
	"encoding/json"
	"fmt"

	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/commands/ci/ciutils"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/pkg/utils"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	"github.com/xanzy/go-gitlab"
)

func NewCmdList(f *cmdutils.Factory) *cobra.Command {
	pipelineListCmd := &cobra.Command{
		Use:   "list [flags]",
		Short: `Get the list of CI/CD pipelines.`,
		Example: heredoc.Doc(`
	glab ci list
	glab ci list --status=failed
	`),
		Long: ``,
		Args: cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			var titleQualifier string

			apiClient, err := f.HttpClient()
			if err != nil {
				return err
			}

			repo, err := f.BaseRepo()
			if err != nil {
				return err
			}

			l := &gitlab.ListProjectPipelinesOptions{}

			format, _ := cmd.Flags().GetString("output")
			jsonOut := format == "json"

			l.Page = 1
			l.PerPage = 30

			if m, _ := cmd.Flags().GetString("status"); m != "" {
				l.Status = gitlab.Ptr(gitlab.BuildStateValue(m))
				titleQualifier = m
			}
			if m, _ := cmd.Flags().GetString("orderBy"); m != "" {
				l.OrderBy = gitlab.Ptr(m)
			}
			if m, _ := cmd.Flags().GetString("sort"); m != "" {
				l.Sort = gitlab.Ptr(m)
			}
			if p, _ := cmd.Flags().GetInt("page"); p != 0 {
				l.Page = p
			}
			if p, _ := cmd.Flags().GetInt("per-page"); p != 0 {
				l.PerPage = p
			}

			pipes, err := api.ListProjectPipelines(apiClient, repo.FullName(), l)
			if err != nil {
				return err
			}

			title := utils.NewListTitle(fmt.Sprintf("%s pipeline", titleQualifier))
			title.RepoName = repo.FullName()
			title.Page = l.Page
			title.CurrentPageTotal = len(pipes)

			if jsonOut {
				pipeListJSON, _ := json.Marshal(pipes)
				fmt.Fprintln(f.IO.StdOut, string(pipeListJSON))
			} else {
				fmt.Fprintf(f.IO.StdOut, "%s\n%s\n", title.Describe(), ciutils.DisplayMultiplePipelines(f.IO, pipes, repo.FullName()))
			}
			return nil
		},
	}
	pipelineListCmd.Flags().StringP("status", "s", "", "Get pipeline with this status. Options: running, pending, success, failed, canceled, skipped, created, manual, waiting_for_resource, preparing, scheduled}")
	pipelineListCmd.Flags().StringP("orderBy", "o", "id", "Order pipelines by this field. Options: id, status, ref, updated_at, user_id.")
	pipelineListCmd.Flags().StringP("sort", "", "desc", "Sort pipelines. Options: asc, desc.")
	pipelineListCmd.Flags().IntP("page", "p", 1, "Page number.")
	pipelineListCmd.Flags().IntP("per-page", "P", 30, "Number of items to list per page.")
	pipelineListCmd.Flags().StringP("output", "F", "text", "Format output. Options: text, json.")

	return pipelineListCmd
}
