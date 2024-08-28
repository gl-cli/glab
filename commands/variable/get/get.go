package get

import (
	"encoding/json"
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	"github.com/xanzy/go-gitlab"
	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/commands/variable/variableutils"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/pkg/iostreams"
)

type GetOps struct {
	HTTPClient func() (*gitlab.Client, error)
	IO         *iostreams.IOStreams
	BaseRepo   func() (glrepo.Interface, error)

	Scope        string
	Key          string
	Group        string
	OutputFormat string
	JSONOutput   bool
}

func NewCmdSet(f *cmdutils.Factory, runE func(opts *GetOps) error) *cobra.Command {
	opts := &GetOps{
		IO: f.IO,
	}

	cmd := &cobra.Command{
		Use:   "get <key>",
		Short: "Get a variable for a project or group.",
		Args:  cobra.RangeArgs(1, 1),
		Example: heredoc.Doc(`
			glab variable get VAR_KEY
			glab variable get -g GROUP VAR_KEY
			glab variable get -s SCOPE VAR_KEY
		`),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			opts.HTTPClient = f.HttpClient
			opts.BaseRepo = f.BaseRepo

			opts.Key = args[0]

			if !variableutils.IsValidKey(opts.Key) {
				err = cmdutils.FlagError{Err: fmt.Errorf("invalid key provided.\n%s", variableutils.ValidKeyMsg)}
				return
			}

			if runE != nil {
				err = runE(opts)
				return
			}
			err = getRun(opts)
			return
		},
	}

	cmd.Flags().StringVarP(&opts.Scope, "scope", "s", "*", "The environment_scope of the variable. Values: all (*), or specific environments.")
	cmd.Flags().StringVarP(&opts.Group, "group", "g", "", "Get variable for a group.")
	cmd.Flags().StringVarP(&opts.OutputFormat, "output", "F", "text", "Format output as: text, json.")
	return cmd
}

func getRun(opts *GetOps) error {
	httpClient, err := opts.HTTPClient()
	if err != nil {
		return err
	}

	var variableValue string

	if opts.Group != "" {
		variable, err := api.GetGroupVariable(httpClient, opts.Group, opts.Key, opts.Scope)
		if err != nil {
			return err
		}
		if opts.OutputFormat == "json" {
			varJSON, _ := json.Marshal(variable)
			fmt.Println(string(varJSON))
		}
		variableValue = variable.Value
	} else {
		baseRepo, err := opts.BaseRepo()
		if err != nil {
			return err
		}

		variable, err := api.GetProjectVariable(httpClient, baseRepo.FullName(), opts.Key, opts.Scope)
		if err != nil {
			return err
		}
		if opts.OutputFormat == "json" {
			varJSON, _ := json.Marshal(variable)
			fmt.Fprintln(opts.IO.StdOut, string(varJSON))
		}
		variableValue = variable.Value
	}

	if opts.OutputFormat != "json" {
		fmt.Fprint(opts.IO.StdOut, variableValue)
	}
	return nil
}
