package get

import (
	"encoding/json"
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/commands/variable/variableutils"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/pkg/iostreams"
)

type options struct {
	httpClient func() (*gitlab.Client, error)
	io         *iostreams.IOStreams
	baseRepo   func() (glrepo.Interface, error)

	scope        string
	key          string
	group        string
	outputFormat string
	jsonOutput   bool
}

func NewCmdGet(f cmdutils.Factory, runE func(opts *options) error) *cobra.Command {
	opts := &options{
		io:         f.IO(),
		httpClient: f.HttpClient,
		baseRepo:   f.BaseRepo,
	}

	cmd := &cobra.Command{
		Use:   "get <key>",
		Short: "Get a variable for a project or group.",
		Args:  cobra.RangeArgs(1, 1),
		Example: heredoc.Doc(`
			$ glab variable get VAR_KEY
			$ glab variable get -g GROUP VAR_KEY
			$ glab variable get -s SCOPE VAR_KEY
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.complete(args)

			if err := opts.validate(); err != nil {
				return err
			}

			if runE != nil {
				return runE(opts)
			}
			return opts.run()
		},
	}

	cmd.Flags().StringVarP(&opts.scope, "scope", "s", "*", "The environment_scope of the variable. Values: all (*), or specific environments.")
	cmd.Flags().StringVarP(&opts.group, "group", "g", "", "Get variable for a group.")
	cmd.Flags().StringVarP(&opts.outputFormat, "output", "F", "text", "Format output as: text, json.")
	return cmd
}

func (o *options) complete(args []string) {
	o.key = args[0]
}

func (o *options) validate() error {
	if !variableutils.IsValidKey(o.key) {
		return cmdutils.FlagError{Err: fmt.Errorf("invalid key provided.\n%s", variableutils.ValidKeyMsg)}
	}

	return nil
}

func (o *options) run() error {
	httpClient, err := o.httpClient()
	if err != nil {
		return err
	}

	var variableValue string

	if o.group != "" {
		variable, err := api.GetGroupVariable(httpClient, o.group, o.key, o.scope)
		if err != nil {
			return err
		}
		if o.outputFormat == "json" {
			varJSON, _ := json.Marshal(variable)
			fmt.Println(string(varJSON))
		}
		variableValue = variable.Value
	} else {
		baseRepo, err := o.baseRepo()
		if err != nil {
			return err
		}

		variable, err := api.GetProjectVariable(httpClient, baseRepo.FullName(), o.key, o.scope)
		if err != nil {
			return err
		}
		if o.outputFormat == "json" {
			varJSON, _ := json.Marshal(variable)
			fmt.Fprintln(o.io.StdOut, string(varJSON))
		}
		variableValue = variable.Value
	}

	if o.outputFormat != "json" {
		fmt.Fprint(o.io.StdOut, variableValue)
	}
	return nil
}
