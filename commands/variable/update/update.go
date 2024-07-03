package update

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

type UpdateOpts struct {
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

func NewCmdSet(f *cmdutils.Factory, runE func(opts *UpdateOpts) error) *cobra.Command {
	opts := &UpdateOpts{
		IO: f.IO,
	}

	cmd := &cobra.Command{
		Use:   "update <key> <value>",
		Short: "Update an existing variable for a project or group.",
		Args:  cobra.RangeArgs(1, 2),
		Example: heredoc.Doc(`
			glab variable update WITH_ARG "some value"
			glab variable update FROM_FLAG -v "some value"
			glab variable update FROM_ENV_WITH_ARG "${ENV_VAR}"
			glab variable update FROM_ENV_WITH_FLAG -v"${ENV_VAR}"
			glab variable update FROM_FILE < secret.txt
			cat file.txt | glab variable update SERVER_TOKEN
			cat token.txt | glab variable update GROUP_TOKEN -g mygroup --scope=prod
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

			if cmd.Flags().Changed("scope") && opts.Group != "" {
				err = cmdutils.FlagError{Err: errors.New("scope is not required for group variables.")}
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
			err = updateRun(opts)
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

func updateRun(opts *UpdateOpts) error {
	c := opts.IO.Color()
	httpClient, err := opts.HTTPClient()
	if err != nil {
		return err
	}

	if opts.Group != "" {
		// update group-level variable
		updateGroupVarOpts := &gitlab.UpdateGroupVariableOptions{
			Value:            gitlab.Ptr(opts.Value),
			VariableType:     gitlab.Ptr(gitlab.VariableTypeValue(opts.Type)),
			Masked:           gitlab.Ptr(opts.Masked),
			Protected:        gitlab.Ptr(opts.Protected),
			Raw:              gitlab.Ptr(opts.Raw),
			EnvironmentScope: gitlab.Ptr(opts.Scope),
		}

		_, err = api.UpdateGroupVariable(httpClient, opts.Group, opts.Key, updateGroupVarOpts)
		if err != nil {
			return err
		}

		fmt.Fprintf(opts.IO.StdOut, "%s Updated variable %s for group %s.\n", c.GreenCheck(), opts.Key, opts.Group)
		return nil
	}

	// update project-level variable
	baseRepo, err := opts.BaseRepo()
	if err != nil {
		return err
	}

	updateProjectVarOpts := &gitlab.UpdateProjectVariableOptions{
		Value:            gitlab.Ptr(opts.Value),
		VariableType:     gitlab.Ptr(gitlab.VariableTypeValue(opts.Type)),
		Masked:           gitlab.Ptr(opts.Masked),
		Protected:        gitlab.Ptr(opts.Protected),
		Raw:              gitlab.Ptr(opts.Raw),
		EnvironmentScope: gitlab.Ptr(opts.Scope),
	}

	_, err = api.UpdateProjectVariable(httpClient, baseRepo.FullName(), opts.Key, updateProjectVarOpts)
	if err != nil {
		return err
	}

	fmt.Fprintf(opts.IO.StdOut, "%s Updated variable %s for project %s with scope %s.\n", c.GreenCheck(), opts.Key, baseRepo.FullName(), opts.Scope)
	return nil
}
