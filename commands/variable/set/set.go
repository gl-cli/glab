package set

import (
	"errors"
	"fmt"

	"gitlab.com/gitlab-org/cli/pkg/iostreams"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	"github.com/xanzy/go-gitlab"
	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/commands/variable/variableutils"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
)

type SetOpts struct {
	HTTPClient func() (*gitlab.Client, error)
	IO         *iostreams.IOStreams
	BaseRepo   func() (glrepo.Interface, error)

	Key       string
	Value     string
	Type      string
	Scope     string
	Protected bool
	Masked    bool
	Raw       bool
	Group     string
}

func NewCmdSet(f *cmdutils.Factory, runE func(opts *SetOpts) error) *cobra.Command {
	opts := &SetOpts{
		IO: f.IO,
	}

	cmd := &cobra.Command{
		Use:     "set <key> <value>",
		Short:   "Create a new variable for a project or group.",
		Aliases: []string{"new", "create"},
		Args:    cobra.RangeArgs(1, 2),
		Example: heredoc.Doc(`
			glab variable set WITH_ARG "some value"
			glab variable set FROM_FLAG -v "some value"
			glab variable set FROM_ENV_WITH_ARG "${ENV_VAR}"
			glab variable set FROM_ENV_WITH_FLAG -v"${ENV_VAR}"
			glab variable set FROM_FILE < secret.txt
			cat file.txt | glab variable set SERVER_TOKEN
			cat token.txt | glab variable set GROUP_TOKEN -g mygroup --scope=prod
		`),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			// Supports repo override
			opts.HTTPClient = f.HttpClient
			opts.BaseRepo = f.BaseRepo

			opts.Key = args[0]

			if !variableutils.IsValidKey(opts.Key) {
				err = cmdutils.FlagError{Err: fmt.Errorf("invalid key provided.\n%s", variableutils.ValidKeyMsg)}
				return
			}

			if opts.Value != "" && len(args) == 2 {
				err = cmdutils.FlagError{Err: errors.New("specify value either by the second positional argument or the --value flag.")}
				return
			}

			opts.Value, err = variableutils.GetValue(opts.Value, opts.IO, args)
			if err != nil {
				return
			}

			if cmd.Flags().Changed("type") {
				if opts.Type != "env_var" && opts.Type != "file" {
					err = cmdutils.FlagError{Err: fmt.Errorf("invalid type: %s. --type must be one of `env_var` or `file`.", opts.Type)}
					return
				}
			}

			if runE != nil {
				err = runE(opts)
				return
			}
			err = setRun(opts)
			return
		},
	}

	cmd.Flags().StringVarP(&opts.Value, "value", "v", "", "The value of a variable.")
	cmd.Flags().StringVarP(&opts.Type, "type", "t", "env_var", "The type of a variable: env_var, file.")
	cmd.Flags().StringVarP(&opts.Scope, "scope", "s", "*", "The environment_scope of the variable. Values: all (*), or specific environments.")
	cmd.Flags().StringVarP(&opts.Group, "group", "g", "", "Set variable for a group.")
	cmd.Flags().BoolVarP(&opts.Masked, "masked", "m", false, "Whether the variable is masked.")
	cmd.Flags().BoolVarP(&opts.Raw, "raw", "r", false, "Whether the variable is treated as a raw string.")
	cmd.Flags().BoolVarP(&opts.Protected, "protected", "p", false, "Whether the variable is protected.")
	return cmd
}

func setRun(opts *SetOpts) error {
	c := opts.IO.Color()
	httpClient, err := opts.HTTPClient()
	if err != nil {
		return err
	}

	if opts.Group != "" {
		// creating group-level variable
		createVarOpts := &gitlab.CreateGroupVariableOptions{
			Key:              gitlab.Ptr(opts.Key),
			Value:            gitlab.Ptr(opts.Value),
			EnvironmentScope: gitlab.Ptr(opts.Scope),
			Masked:           gitlab.Ptr(opts.Masked),
			Protected:        gitlab.Ptr(opts.Protected),
			VariableType:     gitlab.Ptr(gitlab.VariableTypeValue(opts.Type)),
			Raw:              gitlab.Ptr(opts.Raw),
		}
		_, err = api.CreateGroupVariable(httpClient, opts.Group, createVarOpts)
		if err != nil {
			return err
		}

		fmt.Fprintf(opts.IO.StdOut, "%s Created variable %s for group %s.\n", c.GreenCheck(), opts.Key, opts.Group)
		return nil
	}

	// creating project-level variable
	baseRepo, err := opts.BaseRepo()
	if err != nil {
		return err
	}
	createVarOpts := &gitlab.CreateProjectVariableOptions{
		Key:              gitlab.Ptr(opts.Key),
		Value:            gitlab.Ptr(opts.Value),
		EnvironmentScope: gitlab.Ptr(opts.Scope),
		Masked:           gitlab.Ptr(opts.Masked),
		Protected:        gitlab.Ptr(opts.Protected),
		VariableType:     gitlab.Ptr(gitlab.VariableTypeValue(opts.Type)),
		Raw:              gitlab.Ptr(opts.Raw),
	}
	_, err = api.CreateProjectVariable(httpClient, baseRepo.FullName(), createVarOpts)
	if err != nil {
		return err
	}

	fmt.Fprintf(opts.IO.StdOut, "%s Created variable %s for %s with scope %s.\n", c.GreenCheck(), opts.Key, baseRepo.FullName(), opts.Scope)
	return nil
}
