package retry

import (
	"fmt"

	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/commands/ci/ciutils"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
)

func NewCmdRetry(f *cmdutils.Factory) *cobra.Command {
	pipelineRetryCmd := &cobra.Command{
		Use:     "retry <job-id>",
		Short:   `Retry a CI/CD job.`,
		Aliases: []string{},
		Example: heredoc.Doc(`
		$ glab ci retry
		# Interactively select a job to retry.

		$ glab ci retry 224356863
		# Retry job with ID 224356863

		$ glab ci retry lint
		# Retry job with the name 'lint'
`),
		Long: ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error

			repo, err := f.BaseRepo()
			if err != nil {
				return err
			}
			apiClient, err := f.HttpClient()
			if err != nil {
				return err
			}

			jobName := ""
			if len(args) != 0 {
				jobName = args[0]
			}
			branch, _ := cmd.Flags().GetString("branch")
			pipelineId, _ := cmd.Flags().GetInt("pipeline-id")

			jobID, err := ciutils.GetJobId(&ciutils.JobInputs{
				JobName:         jobName,
				Branch:          branch,
				PipelineId:      pipelineId,
				SelectionPrompt: "Select pipeline job to retry:",
			}, &ciutils.JobOptions{
				ApiClient: apiClient,
				IO:        f.IO,
				Repo:      repo,
			})
			if err != nil {
				fmt.Fprintln(f.IO.StdErr, "invalid job ID:", args[0])
				return err
			}

			if jobID == 0 {
				return nil
			}

			job, err := api.RetryPipelineJob(apiClient, jobID, repo.FullName())
			if err != nil {
				return cmdutils.WrapError(err, fmt.Sprintf("Could not retry job with ID: %d", jobID))
			}
			fmt.Fprintln(f.IO.StdOut, "Retried job (ID:", job.ID, "), status:", job.Status, ", ref:", job.Ref, ", weburl: ", job.WebURL, ")")

			return nil
		},
	}

	pipelineRetryCmd.Flags().StringP("branch", "b", "", "The branch to search for the job. Default: current branch.")
	pipelineRetryCmd.Flags().IntP("pipeline-id", "p", 0, "The pipeline ID to search for the job.")
	return pipelineRetryCmd
}
