package status

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/xanzy/go-gitlab"
	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/pkg/git"
	"gitlab.com/gitlab-org/cli/pkg/tableprinter"

	"github.com/MakeNowJust/heredoc"
	"github.com/spf13/cobra"
)

type PipelineMergedResponse struct {
	*gitlab.Pipeline
	Jobs      []*gitlab.Job              `json:"jobs"`
	Variables []*gitlab.PipelineVariable `json:"variables"`
}

func NewCmdGet(f *cmdutils.Factory) *cobra.Command {
	pipelineGetCmd := &cobra.Command{
		Use:     "get [flags]",
		Short:   `Get JSON of a running CI/CD pipeline on the current or other specified branch`,
		Aliases: []string{"stats"},
		Example: heredoc.Doc(`
	glab ci get
	glab ci -R some/project -p 12345
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

			// Parse arguments into local vars
			branch, _ := cmd.Flags().GetString("branch")
			pipelineId, _ := cmd.Flags().GetInt("pipeline-id")

			if branch == "" {
				branch, err = git.CurrentBranch()
				if err != nil {
					return err
				}
			}
			pipeline, err := api.GetPipeline(apiClient, pipelineId, nil, repo.FullName())
			if err != nil {
				redCheck := c.Red("✘")
				fmt.Fprintf(f.IO.StdOut, "%s No pipelines running or available on branch: %s\n", redCheck, branch)
				return err
			}

			jobs, err := api.GetPipelineJobs(apiClient, pipelineId, repo.FullName())
			if err != nil {
				return err
			}

			variables, err := api.GetPipelineVariables(apiClient, pipelineId, nil, repo.FullName())
			if err != nil {
				return err
			}

			mergedPipelineObject := &PipelineMergedResponse{
				Pipeline:  pipeline,
				Jobs:      jobs,
				Variables: variables,
			}

			outputFormat, _ := cmd.Flags().GetString("output-format")
			if outputFormat == "json" {
				printJSON(*mergedPipelineObject)
			} else {
				printTable(*mergedPipelineObject)
			}

			return nil
		},
	}

	pipelineGetCmd.Flags().StringP("branch", "b", "", "Check pipeline status for a branch. (Default is current branch)")
	pipelineGetCmd.Flags().IntP("pipeline-id", "p", 0, "Provide pipeline ID")
	pipelineGetCmd.Flags().StringP("output-format", "o", "text", "Format output as: text, json")

	return pipelineGetCmd
}

func printJSON(p PipelineMergedResponse) {
	JSONStr, _ := json.Marshal(p)
	fmt.Println(string(JSONStr))
}

func printTable(p PipelineMergedResponse) {
	fmt.Print("# Pipeline:\n")
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
	fmt.Println(pipelineTable.String())

	fmt.Print("# Jobs:\n")
	jobTable := tableprinter.NewTablePrinter()
	for _, j := range p.Jobs {
		j := j
		jobTable.AddRow(j.Name+":", j.Status)
	}
	fmt.Println(jobTable.String())

	fmt.Print("# Variables:\n")
	varTable := tableprinter.NewTablePrinter()
	for _, v := range p.Variables {
		varTable.AddRow(v.Key+":", v.Value)
	}
	fmt.Println(varTable.String())
}
