package run_trig

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/commands/ci/ciutils"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	"github.com/xanzy/go-gitlab"
)

var envVariables = []string{}

func parseVarArg(s string) (key string, val string, err error) {
	// From https://pkg.go.dev/strings#Split:
	//
	// > If s does not contain sep and sep is not empty,
	// > Split returns a slice of length 1 whose only element is s.
	//
	// Therefore, the function will always return a slice of min length 1.
	v := strings.SplitN(s, ":", 2)
	if len(v) == 1 {
		return "", "", fmt.Errorf("invalid argument structure")
	}
	return v[0], v[1], nil
}

func NewCmdRunTrig(f *cmdutils.Factory) *cobra.Command {
	pipelineRunCmd := &cobra.Command{
		Use:     "run-trig [flags]",
		Short:   `Run a CI/CD pipeline trigger.`,
		Aliases: []string{"run-trig"},
		Example: heredoc.Doc(`
	glab ci run-trig -t xxxx
	glab ci run-trig -t xxxx -b main
	glab ci run-trig -t xxxx -b main --variables key1:val1
	glab ci run-trig -t xxxx -b main --variables key1:val1,key2:val2
	glab ci run-trig -t xxxx -b main --variables key1:val1 --variables key2:val2
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

			c := &gitlab.RunPipelineTriggerOptions{
				Variables: make(map[string]string),
			}

			if customPipelineVars, _ := cmd.Flags().GetStringSlice("variables"); len(customPipelineVars) > 0 {
				for _, v := range customPipelineVars {
					key, val, err := parseVarArg(v)
					if err != nil {
						return fmt.Errorf("parsing pipeline variable. Expected format KEY:VALUE: %w", err)
					}
					c.Variables[key] = val
				}
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
				// `ci run-trig` is running out of a git repo
				fmt.Fprintln(f.IO.StdOut, "not in a Git repository. Using repository argument.")
				c.Ref = gitlab.Ptr(ciutils.GetDefaultBranch(f))
			}

			token, err := cmd.Flags().GetString("token")
			if err != nil {
				return err
			}
			if token == "" {
				token = os.Getenv("CI_JOB_TOKEN")
			}
			if token == "" {
				return errors.New("`--token` parameter can be omitted only if `CI_JOB_TOKEN` environment variable is set.")
			}
			c.Token = &token

			pipe, err := api.RunPipelineTrigger(apiClient, repo.FullName(), c)
			if err != nil {
				return err
			}

			fmt.Fprintln(f.IO.StdOut, "Created pipeline (ID:", pipe.ID, "), status:", pipe.Status, ", ref:", pipe.Ref, ", weburl: ", pipe.WebURL, ")")
			return nil
		},
	}
	pipelineRunCmd.Flags().StringP("token", "t", "", "Pipeline trigger token. Can be omitted only if the `CI_JOB_TOKEN` environment variable is set.")
	pipelineRunCmd.Flags().StringP("branch", "b", "", "Create pipeline on branch or reference <string>.")
	pipelineRunCmd.Flags().StringSliceVarP(&envVariables, "variables", "", []string{}, "Pass variables to pipeline in the format <key>:<value>.")

	return pipelineRunCmd
}
