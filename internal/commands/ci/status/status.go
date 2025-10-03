package status

import (
	"fmt"
	"time"

	"gitlab.com/gitlab-org/cli/internal/mcpannotations"

	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/commands/ci/ciutils"
	"gitlab.com/gitlab-org/cli/internal/commands/mr/mrutils"
	"gitlab.com/gitlab-org/cli/internal/dbg"
	"gitlab.com/gitlab-org/cli/internal/utils"

	"github.com/AlecAivazis/survey/v2"
	"github.com/MakeNowJust/heredoc/v2"
	"github.com/gosuri/uilive"
	"github.com/spf13/cobra"
)

func NewCmdStatus(f cmdutils.Factory) *cobra.Command {
	pipelineStatusCmd := &cobra.Command{
		Use:     "status [flags]",
		Short:   `View a running CI/CD pipeline on current or other branch specified.`,
		Aliases: []string{"stats"},
		Example: heredoc.Doc(`
		       $ glab ci status --live

		       # A more compact view
		       $ glab ci status --compact

		       # Get the pipeline for the main branch
		       $ glab ci status --branch=main

		       # Get the pipeline for the current branch
		       $ glab ci status
	       `),
		Long: ``,
		Args: cobra.ExactArgs(0),
		Annotations: map[string]string{
			mcpannotations.Safe: "true",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			c := f.IO().Color()

			client, err := f.GitLabClient()
			if err != nil {
				return err
			}

			branch, _ := cmd.Flags().GetString("branch")
			live, _ := cmd.Flags().GetBool("live")
			compact, _ := cmd.Flags().GetBool("compact")
			repo, err := f.BaseRepo()
			if err != nil {
				return err
			}
			repoName := repo.FullName()
			dbg.Debug("Repository:", repoName)

			// Get the correct branch name using the utility function
			branch = ciutils.GetBranch(branch, func() (string, error) {
				return f.Branch()
			}, repo, client)
			dbg.Debug("Using branch:", branch)

			// Use fallback logic for robust pipeline lookup
			runningPipeline, err := getPipelineWithFallback(client, f, repoName, branch)
			if err != nil {
				redCheck := c.Red("✘")
				fmt.Fprintf(f.IO().StdOut, "%s %v\n", redCheck, err)
				return err
			}

			writer := uilive.New()

			// start listening for updates and render
			writer.Start()
			defer writer.Stop()
			for {
				jobs, err := gitlab.ScanAndCollect(func(p gitlab.PaginationOptionFunc) ([]*gitlab.Job, *gitlab.Response, error) {
					return client.Jobs.ListPipelineJobs(repoName, runningPipeline.ID, &gitlab.ListJobsOptions{ListOptions: gitlab.ListOptions{PerPage: 100}}, p)
				})
				if err != nil {
					return err
				}
				for _, job := range jobs {
					end := time.Now()
					if job.FinishedAt != nil {
						end = *job.FinishedAt
					}
					var duration string
					if job.StartedAt != nil {
						duration = utils.FmtDuration(end.Sub(*job.StartedAt))
					} else {
						duration = "not started"
					}
					var status string
					switch s := job.Status; s {
					case "failed":
						if job.AllowFailure {
							status = c.Yellow(s)
						} else {
							status = c.Red(s)
						}
					case "success":
						status = c.Green(s)
					default:
						status = c.Gray(s)
					}
					if compact {
						fmt.Fprintf(writer, "(%s) • %s [%s]\n", status, job.Name, job.Stage)
					} else {
						fmt.Fprintf(writer, "(%s) • %s\t%s\t\t%s\n", status, c.Gray(duration), job.Stage, job.Name)
					}
				}

				if !compact {
					fmt.Fprintf(writer.Newline(), "\n%s\n", runningPipeline.WebURL)
					fmt.Fprintf(writer.Newline(), "SHA: %s\n", runningPipeline.SHA)
				}
				fmt.Fprintf(writer.Newline(), "Pipeline state: %s\n\n", runningPipeline.Status)

				if (runningPipeline.Status == "pending" || runningPipeline.Status == "running") && live {
					// Use fallback logic for live updates
					updatedPipeline, err := getPipelineWithFallback(client, f, repoName, branch)
					if err != nil {
						// Final fallback: refresh current pipeline by ID
						updatedPipeline, _, err = client.Pipelines.GetPipeline(repoName, runningPipeline.ID)
						if err != nil {
							return err
						}
					}
					runningPipeline = updatedPipeline
				} else if f.IO().IsInputTTY() && f.IO().PromptEnabled() {
					prompt := &survey.Select{
						Message: "Choose an action:",
						Options: []string{"View logs", "Retry", "Exit"},
						Default: "Exit",
					}
					var answer string
					_ = survey.AskOne(prompt, &answer)
					switch answer {
					case "View logs":
						return ciutils.TraceJob(&ciutils.JobInputs{
							Branch: branch,
						}, &ciutils.JobOptions{
							Repo:   repo,
							Client: client,
							IO:     f.IO(),
						})
					case "Retry":
						_, _, err := client.Pipelines.RetryPipelineBuild(repoName, runningPipeline.ID)
						if err != nil {
							return err
						}
						updatedPipeline, err := getPipelineWithFallback(client, f, repoName, branch)
						if err != nil {
							// Fallback: refresh by pipeline ID if MR lookup fails
							updatedPipeline, _, err = client.Pipelines.GetPipeline(repoName, runningPipeline.ID)
							if err != nil {
								return err
							}
						}
						runningPipeline = updatedPipeline
					default:
						goto exitLoop
					}
				} else {
					break
				}
			}
		exitLoop:
			if runningPipeline.Status == "failed" {
				return cmdutils.SilentError
			}
			return nil
		},
	}

	pipelineStatusCmd.Flags().BoolP("live", "l", false, "Show status in real time until the pipeline ends.")
	pipelineStatusCmd.Flags().BoolP("compact", "c", false, "Show status in compact format.")
	pipelineStatusCmd.Flags().StringP("branch", "b", "", "Check pipeline status for a branch. (default current branch)")

	return pipelineStatusCmd
}

func getPipelineWithFallback(client *gitlab.Client, f cmdutils.Factory, repoName, branch string) (*gitlab.Pipeline, error) {
	// First try: Get pipeline by branch name
	pipeline, _, err := client.Pipelines.GetLatestPipeline(repoName, &gitlab.GetLatestPipelineOptions{Ref: gitlab.Ptr(branch)})
	if err == nil {
		return pipeline, nil
	}

	// Fallback: Look for MR pipeline
	mr, _, mrErr := mrutils.MRFromArgs(f, []string{}, "any")
	if mr == nil || mrErr != nil {
		return nil, fmt.Errorf("no pipeline found for branch %s and no associated merge request found. Branch may not have an open MR", branch)
	}

	if mr.HeadPipeline == nil {
		return nil, fmt.Errorf("no pipeline found. It might not exist yet. If this problem continues, check your pipeline configuration.")
	}

	// Get the full pipeline details using the MR's head pipeline ID
	pipeline, _, pipelineErr := client.Pipelines.GetPipeline(repoName, mr.HeadPipeline.ID)
	if pipelineErr != nil {
		return nil, pipelineErr
	}

	return pipeline, nil
}
