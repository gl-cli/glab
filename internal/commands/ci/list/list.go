package list

import (
	"encoding/json"
	"fmt"
	"time"

	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/commands/ci/ciutils"
	"gitlab.com/gitlab-org/cli/internal/utils"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

func NewCmdList(f cmdutils.Factory) *cobra.Command {
	pipelineListCmd := &cobra.Command{
		Use:   "list [flags]",
		Short: `Get the list of CI/CD pipelines.`,
		Example: heredoc.Doc(`
			$ glab ci list
			$ glab ci list --status=failed
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

			l := &gitlab.ListProjectPipelinesOptions{
				ListOptions: gitlab.ListOptions{
					Page:    1,
					PerPage: 30,
				},
			}

			format, _ := cmd.Flags().GetString("output")
			jsonOut := format == "json"

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
			if m, _ := cmd.Flags().GetString("ref"); m != "" {
				l.Ref = gitlab.Ptr(m)
			}
			if m, _ := cmd.Flags().GetString("scope"); m != "" {
				l.Scope = gitlab.Ptr(m)
			}
			if m, _ := cmd.Flags().GetString("source"); m != "" {
				l.Source = gitlab.Ptr(m)
			}
			if m, _ := cmd.Flags().GetString("sha"); m != "" {
				l.SHA = gitlab.Ptr(m)
			}
			if m, _ := cmd.Flags().GetBool("yaml-errors"); m {
				l.YamlErrors = gitlab.Ptr(true)
			}
			if m, _ := cmd.Flags().GetString("name"); m != "" {
				l.Name = gitlab.Ptr(m)
			}
			if m, _ := cmd.Flags().GetString("username"); m != "" {
				l.Username = gitlab.Ptr(m)
			}
			if m, _ := cmd.Flags().GetString("updated-after"); m != "" {
				updatedAfterTime, err := time.Parse("2006-01-02T15:04:05Z", m)
				if err != nil {
					return err
				}
				l.UpdatedAfter = gitlab.Ptr(updatedAfterTime)
			}
			if m, _ := cmd.Flags().GetString("updated-before"); m != "" {
				updatedBeforeTime, err := time.Parse("2006-01-02T15:04:05Z", m)
				if err != nil {
					return err
				}
				l.UpdatedBefore = gitlab.Ptr(updatedBeforeTime)
			}

			pipes, _, err := apiClient.Pipelines.ListProjectPipelines(repo.FullName(), l)
			if err != nil {
				return err
			}

			title := utils.NewListTitle(fmt.Sprintf("%s pipeline", titleQualifier))
			title.RepoName = repo.FullName()
			title.Page = l.Page
			title.CurrentPageTotal = len(pipes)

			if jsonOut {
				pipeListJSON, _ := json.Marshal(pipes)
				fmt.Fprintln(f.IO().StdOut, string(pipeListJSON))
			} else {
				fmt.Fprintf(f.IO().StdOut, "%s\n%s\n", title.Describe(), ciutils.DisplayMultiplePipelines(f.IO(), pipes, repo.FullName()))
			}
			return nil
		},
	}
	pipelineListCmd.Flags().StringP("status", "s", "", "Get pipeline with this status. Options: running, pending, success, failed, canceled, skipped, created, manual, waiting_for_resource, preparing, scheduled")
	pipelineListCmd.Flags().StringP("orderBy", "o", "id", "Order pipelines by this field. Options: id, status, ref, updated_at, user_id.")
	pipelineListCmd.Flags().StringP("sort", "", "desc", "Sort pipelines. Options: asc, desc.")
	pipelineListCmd.Flags().IntP("page", "p", 1, "Page number.")
	pipelineListCmd.Flags().IntP("per-page", "P", 30, "Number of items to list per page.")
	pipelineListCmd.Flags().StringP("output", "F", "text", "Format output. Options: text, json.")
	pipelineListCmd.Flags().StringP("ref", "r", "", "Return only pipelines for given ref.")
	pipelineListCmd.Flags().String("scope", "", "Return only pipelines with the given scope: {running|pending|finished|branches|tags}")
	pipelineListCmd.Flags().String("source", "", "Return only pipelines triggered via the given source. See https://docs.gitlab.com/ci/jobs/job_rules/#ci_pipeline_source-predefined-variable for full list. Commonly used options: {merge_request_event|parent_pipeline|pipeline|push|trigger}")
	pipelineListCmd.Flags().String("sha", "", "Return only pipelines with the given SHA.")
	pipelineListCmd.Flags().BoolP("yaml-errors", "y", false, "Return only pipelines with invalid configurations.")
	pipelineListCmd.Flags().StringP("name", "n", "", "Return only pipelines with the given name.")
	pipelineListCmd.Flags().StringP("username", "u", "", "Return only pipelines triggered by the given username.")
	pipelineListCmd.Flags().StringP("updated-before", "b", "", "Return only pipelines updated before the specified date. Expected in ISO 8601 format (2019-03-15T08:00:00Z).")
	pipelineListCmd.Flags().StringP("updated-after", "a", "", "Return only pipelines updated after the specified date. Expected in ISO 8601 format (2019-03-15T08:00:00Z).")

	return pipelineListCmd
}
