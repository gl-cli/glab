package run

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/commands/ci/ciutils"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/pkg/utils"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

var (
	PipelineVarTypeEnv  = gitlab.EnvVariableType
	PipelineVarTypeFile = gitlab.FileVariableType
	envVariables        = []string{}
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
	pvar.VariableType = &PipelineVarTypeEnv
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
	pvar.VariableType = &PipelineVarTypeFile
	pvar.Value = &content
	return pvar, nil
}

func NewCmdRun(f *cmdutils.Factory) *cobra.Command {
	openInBrowser := false

	pipelineRunCmd := &cobra.Command{
		Use:     "run [flags]",
		Short:   `Create or run a new CI/CD pipeline.`,
		Aliases: []string{"create"},
		Example: heredoc.Doc(`
			- glab ci run
			- glab ci run --variables \"key1:value,with,comma\"
			- glab ci run -b main
			- glab ci run -b main --variables-env key1:val1
			- glab ci run -b main --variables-env key1:val1,key2:val2
			- glab ci run -b main --variables-env key1:val1 --variables-env key2:val2
			- glab ci run -b main --variables-file MYKEY:file1 --variables KEY2:some_value
		`),
		Long: ``,
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

			pipelineVars := []*gitlab.PipelineVariableOptions{}

			if customPipelineVars, _ := cmd.Flags().GetStringSlice("variables-env"); len(customPipelineVars) > 0 {
				for _, v := range customPipelineVars {
					pvar, err := extractEnvVar(v)
					if err != nil {
						return fmt.Errorf("parsing pipeline variable expected format KEY:VALUE: %w", err)
					}
					pipelineVars = append(pipelineVars, pvar)
				}
			}

			if customPipelineFileVars, _ := cmd.Flags().GetStringSlice("variables-file"); len(customPipelineFileVars) > 0 {
				for _, v := range customPipelineFileVars {
					pvar, err := extractFileVar(v)
					if err != nil {
						return fmt.Errorf("parsing pipeline variable. Expected format KEY:FILENAME: %w", err)
					}
					pipelineVars = append(pipelineVars, pvar)
				}
			}

			vf, err := cmd.Flags().GetString("variables-from")
			if err != nil {
				return err
			}
			if vf != "" {
				b, err := os.ReadFile(vf)
				if err != nil {
					// Return the error encountered
					return fmt.Errorf("opening variable file: %s", vf)
				}
				var result []*gitlab.PipelineVariableOptions
				err = json.Unmarshal(b, &result)
				if err != nil {
					return fmt.Errorf("loading pipeline values: %w", err)
				}
				pipelineVars = append(pipelineVars, result...)
			}

			c := &gitlab.CreatePipelineOptions{
				Variables: &pipelineVars,
			}

			branch, err := cmd.Flags().GetString("branch")
			if err != nil {
				return err
			}
			if branch != "" {
				c.Ref = gitlab.Ptr(branch)
			} else if currentBranch, err := f.Branch(); err == nil {
				c.Ref = gitlab.Ptr(currentBranch)
			} else {
				// `ci run` is running out of a git repo
				fmt.Fprintln(f.IO.StdOut, "not in a Git repository. Using repository argument.")
				c.Ref = gitlab.Ptr(ciutils.GetDefaultBranch(f))
			}

			pipe, err := api.CreatePipeline(apiClient, repo.FullName(), c)
			if err != nil {
				return err
			}

			if openInBrowser { // open in browser if --web flag is specified
				webURL := pipe.WebURL

				if f.IO.IsOutputTTY() {
					fmt.Fprintf(f.IO.StdErr, "Opening %s in your browser.\n", utils.DisplayURL(webURL))
				}

				cfg, err := f.Config()
				if err != nil {
					return err
				}

				browser, _ := cfg.Get(repo.RepoHost(), "browser")
				return utils.OpenInBrowser(webURL, browser)
			}

			output := fmt.Sprintf("Created pipeline (id: %d), status: %s, ref: %s, weburl: %s", pipe.ID, pipe.Status, pipe.Ref, pipe.WebURL)
			fmt.Fprintln(f.IO.StdOut, output)
			return nil
		},
	}
	pipelineRunCmd.Flags().StringP("branch", "b", "", "Create pipeline on branch/ref <string>.")
	pipelineRunCmd.Flags().StringSliceVarP(&envVariables, "variables", "", []string{}, "Pass variables to pipeline in format <key>:<value>.")
	pipelineRunCmd.Flags().StringSliceVarP(&envVariables, "variables-env", "", []string{}, "Pass variables to pipeline in format <key>:<value>.")
	pipelineRunCmd.Flags().StringSliceP("variables-file", "", []string{}, "Pass file contents as a file variable to pipeline in format <key>:<filename>.")
	pipelineRunCmd.Flags().StringP("variables-from", "f", "", "JSON file containing variables for pipeline execution.")
	pipelineRunCmd.Flags().BoolVarP(&openInBrowser, "web", "w", false, "Open pipeline in a browser. Uses default browser, or browser specified in BROWSER environment variable.")

	return pipelineRunCmd
}
