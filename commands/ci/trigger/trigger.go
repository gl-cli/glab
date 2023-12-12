package trigger

import (
	"fmt"

	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/commands/ci/ciutils"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"

	"github.com/MakeNowJust/heredoc"
	"github.com/spf13/cobra"
)

func NewCmdTrigger(f *cmdutils.Factory) *cobra.Command {
	pipelineTriggerCmd := &cobra.Command{
		Use:     "trigger <job-id>",
		Short:   `Trigger a manual CI/CD job.`,
		Aliases: []string{},
		Example: heredoc.Doc(`
		$ glab ci trigger
		# Tnteractively select a job to Trigger

		$ glab ci trigger 224356863
		# Trigger manual job with id 224356863

		$ glab ci trigger lint
		# Trigger manual job with name lint
`),
		Long: ``,
		Args: cobra.ExactArgs(1),
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
				JobName:    jobName,
				Branch:     branch,
				PipelineId: pipelineId,
			}, &ciutils.JobOptions{
				ApiClient: apiClient,
				IO:        f.IO,
				Repo:      repo,
			})
			if err != nil {
				fmt.Fprintln(f.IO.StdErr, "invalid job ID:", args[0])
				return err
			}

			job, err := api.PlayPipelineJob(apiClient, jobID, repo.FullName())
			if err != nil {
				return cmdutils.WrapError(err, fmt.Sprintf("Could not trigger job with ID: %d", jobID))
			}
			fmt.Fprintln(f.IO.StdOut, "Triggered job (ID:", job.ID, "), status:", job.Status, ", ref:", job.Ref, ", weburl: ", job.WebURL, ")")

			return nil
		},
	}

	pipelineTriggerCmd.Flags().StringP("branch", "b", "", "The branch to search for the job. (Default: current branch)")
	pipelineTriggerCmd.Flags().IntP("pipeline-id", "p", 0, "The pipeline ID to search for the job.")
	return pipelineTriggerCmd
}
