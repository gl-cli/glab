package run

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/commands/ci/ciutils"
	"gitlab.com/gitlab-org/cli/internal/commands/mr/mrutils"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/internal/utils"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

func parseVarArg(s string) (*gitlab.PipelineVariableOptions, error) {
	// From https://pkg.go.dev/strings#Split:
	//
	// > If s does not contain sep and sep is not empty,
	// > Split returns a slice of length 1 whose only element is s.
	//
	// Therefore, the function will always return a slice of min length 1.
	v := strings.SplitN(s, ":", 2)
	if len(v) == 1 {
		return nil, fmt.Errorf("invalid argument structure")
	}
	return &gitlab.PipelineVariableOptions{
		Key:   &v[0],
		Value: &v[1],
	}, nil
}

func extractEnvVar(s string) (*gitlab.PipelineVariableOptions, error) {
	pvar, err := parseVarArg(s)
	if err != nil {
		return nil, err
	}
	pvar.VariableType = gitlab.Ptr(gitlab.EnvVariableType)
	return pvar, nil
}

func extractFileVar(s string) (*gitlab.PipelineVariableOptions, error) {
	pvar, err := parseVarArg(s)
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(*pvar.Value)
	if err != nil {
		return nil, err
	}
	content := string(b)
	pvar.VariableType = gitlab.Ptr(gitlab.FileVariableType)
	pvar.Value = &content
	return pvar, nil
}

type PipelineData struct {
	ID     int    `json:"id"`
	Status string `json:"status"`
	Ref    string `json:"ref"`
	WebURL string `json:"web_url"`
}

func createPipeline(cmd *cobra.Command, c *gitlab.CreatePipelineOptions, f cmdutils.Factory, apiClient *gitlab.Client, repo glrepo.Interface, mr bool) (*PipelineData, error) {
	branch, err := resolveBranch(cmd, f)
	if err != nil {
		return nil, err
	}
	if mr {
		pipe, err := createMrPipeline(branch, f, apiClient, repo)
		if err != nil {
			return nil, fmt.Errorf("could not create mr pipeline for branch %s: %v", branch, err)
		}
		return &PipelineData{
			ID:     pipe.ID,
			Status: pipe.Status,
			Ref:    pipe.Ref,
			WebURL: pipe.WebURL,
		}, nil
	}
	pipe, err := createBranchPipeline(branch, c, apiClient, repo)
	if err != nil {
		return nil, fmt.Errorf("could not create branch pipeline for branch %s: %v", branch, err)
	}
	return &PipelineData{
		ID:     pipe.ID,
		Status: pipe.Status,
		Ref:    pipe.Ref,
		WebURL: pipe.WebURL,
	}, nil
}

func createBranchPipeline(branch string, c *gitlab.CreatePipelineOptions, apiClient *gitlab.Client, repo glrepo.Interface) (*gitlab.Pipeline, error) {
	c.Ref = gitlab.Ptr(branch)
	pipe, _, err := apiClient.Pipelines.CreatePipeline(repo.FullName(), c)
	return pipe, err
}

func resolveBranch(cmd *cobra.Command, f cmdutils.Factory) (string, error) {
	br, err := cmd.Flags().GetString("branch")
	if err != nil {
		return "", err
	}
	var branch string
	if br != "" {
		branch = br
	} else if currentBranch, err := f.Branch(); err == nil {
		branch = currentBranch
	} else {
		// `ci run` is running out of a git repo
		fmt.Fprintln(f.IO().StdOut, "not in a Git repository. Using repository argument.")
		branch = ciutils.GetDefaultBranch(f)
	}
	return branch, nil
}

func createMrPipeline(branch string, f cmdutils.Factory, apiClient *gitlab.Client, repo glrepo.Interface) (*gitlab.PipelineInfo, error) {
	mr, err := mrutils.GetMRForBranch(
		apiClient,
		mrutils.MrOptions{
			BaseRepo: repo, Branch: branch, State: "opened", PromptEnabled: f.IO().PromptEnabled(),
		},
	)
	if err != nil {
		return nil, err
	}

	pipe, _, err := apiClient.MergeRequests.CreateMergeRequestPipeline(repo.FullName(), mr.IID)
	if err != nil {
		return nil, err
	}
	return pipe, nil
}

func resolvePipelineVars(cmd *cobra.Command) ([]*gitlab.PipelineVariableOptions, error) {
	pipelineVars := []*gitlab.PipelineVariableOptions{}
	if customPipelineVars, _ := cmd.Flags().GetStringSlice("variables-env"); len(customPipelineVars) > 0 {
		for _, v := range customPipelineVars {
			pvar, err := extractEnvVar(v)
			if err != nil {
				return nil, fmt.Errorf("parsing pipeline variable. Expected format KEY:VALUE: %w", err)
			}
			pipelineVars = append(pipelineVars, pvar)
		}
	}

	if customPipelineFileVars, _ := cmd.Flags().GetStringSlice("variables-file"); len(customPipelineFileVars) > 0 {
		for _, v := range customPipelineFileVars {
			pvar, err := extractFileVar(v)
			if err != nil {
				return nil, fmt.Errorf("parsing pipeline variable. Expected format KEY:FILENAME: %w", err)
			}
			pipelineVars = append(pipelineVars, pvar)
		}
	}

	vf, err := cmd.Flags().GetString("variables-from")
	if err != nil {
		return nil, err
	}

	if vf != "" {
		b, err := os.ReadFile(vf)
		if err != nil {
			return nil, fmt.Errorf("opening variable file: %s", vf)
		}
		var result []*gitlab.PipelineVariableOptions
		err = json.Unmarshal(b, &result)
		if err != nil {
			return nil, fmt.Errorf("loading pipeline values: %w", err)
		}
		pipelineVars = append(pipelineVars, result...)
	}

	return pipelineVars, nil
}

func NewCmdRun(f cmdutils.Factory) *cobra.Command {
	openInBrowser := false
	mr := false

	pipelineRunCmd := &cobra.Command{
		Use:     "run [flags]",
		Short:   `Create or run a new CI/CD pipeline.`,
		Aliases: []string{"create"},
		Example: heredoc.Doc(`
			$ glab ci run
			$ glab ci run --variables \"key1:value,with,comma\"
			$ glab ci run -b main
			$ glab ci run --web
			$ glab ci run -b main --variables-env key1:val1
			$ glab ci run -b main --variables-env key1:val1,key2:val2
			$ glab ci run -b main --variables-env key1:val1 --variables-env key2:val2
			$ glab ci run -b main --variables-file MYKEY:file1 --variables KEY2:some_value
			$ glab ci run --mr

			// For an example of 'glab ci run -f' with a variables file, see
			// [Run a CI/CD pipeline with variables from a file](https://docs.gitlab.com/editor_extensions/gitlab_cli/#run-a-cicd-pipeline-with-variables-from-a-file)
			// in the GitLab documentation.
			`),

		Long: "The `--branch` " + `option is available for all pipeline types.

The options for variables are incompatible with merge request pipelines.
If used with merge request pipelines, the command fails with a message like ` + "`ERROR: if any flags in the group [output output-format] are set none of the others can be`" + `
`,
		Args: cobra.ExactArgs(0),
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

			pipelineVars, err := resolvePipelineVars(cmd)
			if err != nil {
				return err
			}

			c := &gitlab.CreatePipelineOptions{
				Variables: &pipelineVars,
			}

			pipe, err := createPipeline(cmd, c, f, apiClient, repo, mr)
			if err != nil {
				return err
			}
			if openInBrowser { // open in browser if --web flag is specified
				webURL := pipe.WebURL

				if f.IO().IsOutputTTY() {
					fmt.Fprintf(f.IO().StdErr, "Opening %s in your browser.\n", utils.DisplayURL(webURL))
				}

				cfg := f.Config()

				browser, _ := cfg.Get(repo.RepoHost(), "browser")
				return utils.OpenInBrowser(webURL, browser)
			}

			output := fmt.Sprintf("Created pipeline (id: %d), status: %s, ref: %s, weburl: %s", pipe.ID, pipe.Status, pipe.Ref, pipe.WebURL)
			fmt.Fprintln(f.IO().StdOut, output)
			return nil
		},
	}
	pipelineRunCmd.Flags().StringP("branch", "b", "", "Create pipeline on branch/ref <string>.")
	pipelineRunCmd.Flags().StringSliceP("variables", "", []string{}, "Pass variables to pipeline in format <key>:<value>. Cannot be used for MR pipelines.")
	pipelineRunCmd.Flags().StringSliceP("variables-env", "", []string{}, "Pass variables to pipeline in format <key>:<value>. Cannot be used for MR pipelines.")
	pipelineRunCmd.Flags().StringSliceP("variables-file", "", []string{}, "Pass file contents as a file variable to pipeline in format <key>:<filename>. Cannot be used for MR pipelines.")
	pipelineRunCmd.Flags().StringP("variables-from", "f", "", "JSON file with variables for pipeline execution. Expects array of hashes, each with at least 'key' and 'value'. Cannot be used for MR pipelines.")
	pipelineRunCmd.Flags().BoolVarP(&openInBrowser, "web", "w", false, "Open pipeline in a browser. Uses default browser, or browser specified in BROWSER environment variable.")
	pipelineRunCmd.Flags().BoolVar(&mr, "mr", false, "Run merge request pipeline instead of branch pipeline.")

	for _, flag := range []string{"variables", "variables-env", "variables-file", "variables-from"} {
		// https://docs.gitlab.com/api/merge_requests/#create-merge-request-pipeline
		// MR pipeline creation API does not accept variables unlike "normal" pipelines
		// https://docs.gitlab.com/api/pipelines/#create-a-new-pipeline
		pipelineRunCmd.MarkFlagsMutuallyExclusive("mr", flag)
	}

	return pipelineRunCmd
}
