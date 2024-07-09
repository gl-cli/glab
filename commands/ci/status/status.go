package status

import (
	"fmt"
	"time"

	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/commands/ci/ciutils"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/pkg/git"
	"gitlab.com/gitlab-org/cli/pkg/utils"

	"github.com/AlecAivazis/survey/v2"
	"github.com/MakeNowJust/heredoc/v2"
	"github.com/gosuri/uilive"
	"github.com/spf13/cobra"
)

func NewCmdStatus(f *cmdutils.Factory) *cobra.Command {
	pipelineStatusCmd := &cobra.Command{
		Use:     "status [flags]",
		Short:   `View a running CI/CD pipeline on current or other branch specified.`,
		Aliases: []string{"stats"},
		Example: heredoc.Doc(`
	glab ci status --live

	# A more compact view
	glab ci status --compact

	# Get the pipeline for the main branch
	glab ci status --branch=main

	# Get the pipeline for the current branch
	glab ci status
	`),
		Long: ``,
		Args: cobra.ExactArgs(0),
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

			branch, _ := cmd.Flags().GetString("branch")
			live, _ := cmd.Flags().GetBool("live")
			compact, _ := cmd.Flags().GetBool("compact")

			if branch == "" {
				branch, err = git.CurrentBranch()
				if err != nil {
					return err
				}
			}
			runningPipeline, err := api.GetLastPipeline(apiClient, repo.FullName(), branch)
			if err != nil {
				redCheck := c.Red("✘")
				fmt.Fprintf(f.IO.StdOut, "%s No pipelines running or available on branch: %s\n", redCheck, branch)
				return err
			}

			isRunning := true
			retry := "Exit"
			writer := uilive.New()

			// start listening for updates and render
			writer.Start()
			defer writer.Stop()
			for isRunning {
				jobs, err := api.GetPipelineJobs(apiClient, runningPipeline.ID, repo.FullName())
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
					// fmt.Println(job.Tag)
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

				// break loop if input or output is a TTY to avoid prompting
				if !(f.IO.IsInputTTY() && f.IO.PromptEnabled()) {
					break
				}
				if (runningPipeline.Status == "pending" || runningPipeline.Status == "running") && live {
					runningPipeline, err = api.GetLastPipeline(apiClient, repo.FullName(), branch)
					if err != nil {
						return err
					}
				} else {
					prompt := &survey.Select{
						Message: "Choose an action:",
						Options: []string{"View logs", "Retry", "Exit"},
						Default: "Exit",
					}
					_ = survey.AskOne(prompt, &retry)
					if retry != "" && retry != "Exit" {
						if retry == "View logs" {
							isRunning = false
						} else {
							_, err = api.RetryPipeline(apiClient, runningPipeline.ID, repo.FullName())
							if err != nil {
								return err
							}
							runningPipeline, err = api.GetLastPipeline(apiClient, repo.FullName(), branch)
							if err != nil {
								return err
							}
							isRunning = true
						}
					} else {
						isRunning = false
					}
				}
				if retry == "View logs" {
					return ciutils.TraceJob(&ciutils.JobInputs{
						Branch: branch,
					}, &ciutils.JobOptions{
						Repo:      repo,
						ApiClient: apiClient,
						IO:        f.IO,
					})
				}
			}
			return nil
		},
	}

	pipelineStatusCmd.Flags().BoolP("live", "l", false, "Show status in real time until the pipeline ends.")
	pipelineStatusCmd.Flags().BoolP("compact", "c", false, "Show status in compact format.")
	pipelineStatusCmd.Flags().StringP("branch", "b", "", "Check pipeline status for a branch. Default: current branch.")

	return pipelineStatusCmd
}
