package trigger

import (
	"fmt"

	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/pkg/git"
	"gitlab.com/gitlab-org/cli/pkg/utils"

	"github.com/MakeNowJust/heredoc"
	"github.com/spf13/cobra"
	"github.com/xanzy/go-gitlab"
)

func NewCmdTrigger(f *cmdutils.Factory) *cobra.Command {
	pipelineTriggerCmd := &cobra.Command{
		Use:     "trigger <job-id>",
		Short:   `Trigger a manual CI/CD job.`,
		Aliases: []string{},
		Example: heredoc.Doc(`
		$ glab ci trigger 224356863
		#=> trigger manual job with id 224356863

		$ glab ci trigger lint
		#=> trigger manual job with name lint
`),
		Long: ``,
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

			jobID := utils.StringToInt(args[0])

			if jobID < 1 {
				jobName := args[0]

				pipelineId, err := cmd.Flags().GetInt("pipeline-id")
				if err != nil || pipelineId == 0 {
					branch, _ := cmd.Flags().GetString("branch")
					if branch == "" {
						branch, err = git.CurrentBranch()
						if err != nil {
							return err
						}
					}
					commit, err := api.GetCommit(apiClient, repo.FullName(), branch)
					if err != nil {
						return err
					}
					pipelineId = commit.LastPipeline.ID
				}

				jobs, _, err := apiClient.Jobs.ListPipelineJobs(repo.FullName(), pipelineId, nil)
				if err != nil {
					return err
				}
				for _, job := range jobs {
					if job.Name == jobName {
						jobID = job.ID
						break
					}
				}
				if jobID < 1 {
					fmt.Fprintln(f.IO.StdErr, "invalid job id:", args[0])
					return cmdutils.SilentError
				}
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

type Jobs []*gitlab.Job

// FindByName returns the first Remote whose name matches the list
func (jobs Jobs) FindByName(name string) (*gitlab.Job, error) {
	for _, job := range jobs {
		if job.Name == name {
			return job, nil
		}
	}
	return nil, cmdutils.SilentError
}
