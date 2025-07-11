package trace

import (
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/commands/ci/ciutils"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
)

func NewCmdTrace(f cmdutils.Factory) *cobra.Command {
	pipelineCITraceCmd := &cobra.Command{
		Use:   "trace [<job-id>] [flags]",
		Short: `Trace a CI/CD job log in real time.`,
		Example: heredoc.Doc(`
			# Interactively select a job to trace
			$ glab ci trace

			# Trace job with ID 224356863
			$ glab ci trace 224356863

			# Trace job with the name 'lint'
			$ glab ci trace lint
		`),
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

			return ciutils.TraceJob(&ciutils.JobInputs{
				JobName:    jobName,
				Branch:     branch,
				PipelineId: pipelineId,
			}, &ciutils.JobOptions{
				ApiClient: apiClient,
				IO:        f.IO(),
				Repo:      repo,
			})
		},
	}

	pipelineCITraceCmd.Flags().StringP("branch", "b", "", "The branch to search for the job. (default current branch)")
	pipelineCITraceCmd.Flags().IntP("pipeline-id", "p", 0, "The pipeline ID to search for the job.")
	return pipelineCITraceCmd
}
