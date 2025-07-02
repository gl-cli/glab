package get

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"

	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/commands/mr/mrutils"
	"gitlab.com/gitlab-org/cli/internal/git"
	"gitlab.com/gitlab-org/cli/internal/tableprinter"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
)

const NoVariablesInPipelineMessage = "No variables found in pipeline."

type PipelineMergedResponse struct {
	*gitlab.Pipeline
	Jobs      []*gitlab.Job              `json:"jobs"`
	Variables []*gitlab.PipelineVariable `json:"variables"`
}

func NewCmdGet(f cmdutils.Factory) *cobra.Command {
	pipelineGetCmd := &cobra.Command{
		Use:     "get [flags]",
		Short:   `Get JSON of a running CI/CD pipeline on the current or other specified branch.`,
		Aliases: []string{"stats"},
		Example: heredoc.Doc(`
			$ glab ci get
			$ glab ci -R some/project -p 12345
		`),
		Long: ``,
		Args: cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			c := f.IO().Color()

			apiClient, err := f.HttpClient()
			if err != nil {
				return err
			}

			repo, err := f.BaseRepo()
			if err != nil {
				return err
			}

			// Parse arguments into local vars
			branch, _ := cmd.Flags().GetString("branch")
			pipelineId, err := cmd.Flags().GetInt("pipeline-id")
			if err != nil {
				return err
			}

			var msgNotFound string
			if pipelineId != 0 {
				msgNotFound = fmt.Sprintf("No pipeline with the given ID: %d", pipelineId)
			} else {
				if branch == "" {
					branch, err = git.CurrentBranch()
					if err != nil {
						return err
					}
				}

				commit, _, err := apiClient.Commits.GetCommit(repo.FullName(), branch, nil)
				if err != nil {
					return err
				}

				// The latest commit on the branch won't work with a merged
				// result pipeline
				if commit.LastPipeline == nil {
					mr, _, err := mrutils.MRFromArgs(f, args, "any")
					if err != nil {
						return err
					}

					if mr.HeadPipeline == nil {
						return fmt.Errorf("no pipeline found. It might not exist yet. If this problem continues, check your pipeline configuration.")
					} else {
						pipelineId = mr.HeadPipeline.ID
					}

				} else {
					pipelineId = commit.LastPipeline.ID
				}
				msgNotFound = fmt.Sprintf("No pipelines running or available on branch: %s", branch)
			}

			pipeline, _, err := apiClient.Pipelines.GetPipeline(repo.FullName(), pipelineId)
			if err != nil {
				redCheck := c.Red("✘")
				fmt.Fprintf(f.IO().StdOut, "%s %s\n", redCheck, msgNotFound)
				return err
			}

			jobs, err := gitlab.ScanAndCollect(func(p gitlab.PaginationOptionFunc) ([]*gitlab.Job, *gitlab.Response, error) {
				return apiClient.Jobs.ListPipelineJobs(repo.FullName(), pipelineId, &gitlab.ListJobsOptions{ListOptions: gitlab.ListOptions{PerPage: 100}}, p)
			})
			if err != nil {
				return err
			}

			showVariables, _ := cmd.Flags().GetBool("with-variables")

			var variables []*gitlab.PipelineVariable
			if showVariables {
				variables, _, err = apiClient.Pipelines.GetPipelineVariables(pipeline.ProjectID, pipelineId)
				if err != nil {
					return err
				}
			}

			mergedPipelineObject := &PipelineMergedResponse{
				Pipeline:  pipeline,
				Jobs:      jobs,
				Variables: variables,
			}

			outputFormat, _ := cmd.Flags().GetString("output-format")
			output, _ := cmd.Flags().GetString("output")
			if output == "json" || outputFormat == "json" {
				printJSON(*mergedPipelineObject, f.IO().StdOut)
			} else {
				showJobDetails, _ := cmd.Flags().GetBool("with-job-details")
				printTable(*mergedPipelineObject, f.IO().StdOut, showJobDetails)
			}

			return nil
		},
	}

	pipelineGetCmd.Flags().StringP("branch", "b", "", "Check pipeline status for a branch. (default current branch)")
	pipelineGetCmd.Flags().IntP("pipeline-id", "p", 0, "Provide pipeline ID.")
	pipelineGetCmd.Flags().StringP("output", "F", "text", "Format output. Options: text, json.")
	pipelineGetCmd.Flags().StringP("output-format", "o", "text", "Use output.")
	_ = pipelineGetCmd.Flags().MarkHidden("output-format")
	_ = pipelineGetCmd.Flags().MarkDeprecated("output-format", "Deprecated. Use 'output' instead.")
	pipelineGetCmd.Flags().BoolP("with-job-details", "d", false, "Show extended job information.")
	pipelineGetCmd.Flags().Bool("with-variables", false, "Show variables in pipeline. Requires the Maintainer role.")

	return pipelineGetCmd
}

func printJSON(p PipelineMergedResponse, dest io.Writer) {
	JSONStr, _ := json.Marshal(p)
	fmt.Fprintln(dest, string(JSONStr))
}

func printTable(p PipelineMergedResponse, dest io.Writer, showJobDetails bool) {
	printPipelineTable(p, dest)

	if showJobDetails {
		printJobTable(p, dest)
	} else {
		printJobText(p, dest)
	}

	printVariables(p, dest)
}

func printPipelineTable(p PipelineMergedResponse, dest io.Writer) {
	fmt.Fprint(dest, "# Pipeline:\n")
	pipelineTable := tableprinter.NewTablePrinter()
	pipelineTable.AddRow("id:", strconv.Itoa(p.ID))
	pipelineTable.AddRow("status:", p.Status)
	pipelineTable.AddRow("source:", p.Source)
	pipelineTable.AddRow("ref:", p.Ref)
	pipelineTable.AddRow("sha:", p.SHA)
	pipelineTable.AddRow("tag:", p.Tag)
	pipelineTable.AddRow("yaml Errors:", p.YamlErrors)
	pipelineTable.AddRow("user:", p.User.Username)
	pipelineTable.AddRow("created:", p.CreatedAt)
	pipelineTable.AddRow("started:", p.StartedAt)
	pipelineTable.AddRow("updated:", p.UpdatedAt)
	fmt.Fprintln(dest, pipelineTable.String())
}

func printJobTable(p PipelineMergedResponse, dest io.Writer) {
	fmt.Fprint(dest, "# Jobs:\n")
	jobTable := tableprinter.NewTablePrinter()
	jobTable.AddRow("ID", "Name", "Status", "Duration", "Failure reason")
	for _, j := range p.Jobs {
		j := j
		jobTable.AddRow(j.ID, j.Name, j.Status, j.Duration, j.FailureReason)
	}
	fmt.Fprintln(dest, jobTable.String())
}

func printJobText(p PipelineMergedResponse, dest io.Writer) {
	fmt.Fprint(dest, "# Jobs:\n")
	jobTable := tableprinter.NewTablePrinter()
	for _, j := range p.Jobs {
		j := j
		jobTable.AddRow(j.Name+":", j.Status)
	}
	fmt.Fprintln(dest, jobTable.String())
}

func printVariables(p PipelineMergedResponse, dest io.Writer) {
	if p.Variables != nil {
		fmt.Fprint(dest, "# Variables:\n")
		if len(p.Variables) == 0 {
			fmt.Fprint(dest, NoVariablesInPipelineMessage)
		}

		varTable := tableprinter.NewTablePrinter()
		for _, v := range p.Variables {
			varTable.AddRow(v.Key+":", v.Value)
		}
		fmt.Fprintln(dest, varTable.String())
	}
}
