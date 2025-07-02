package set

import (
	"errors"
	"fmt"

	"gitlab.com/gitlab-org/cli/internal/iostreams"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/cli/internal/api"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/commands/variable/variableutils"
	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
)

type options struct {
	apiClient func(repoHost string, cfg config.Config) (*api.Client, error)
	config    config.Config
	io        *iostreams.IOStreams
	baseRepo  func() (glrepo.Interface, error)

	key         string
	value       string
	typ         string
	scope       string
	protected   bool
	masked      bool
	raw         bool
	group       string
	description string
}

func NewCmdSet(f cmdutils.Factory, runE func(opts *options) error) *cobra.Command {
	opts := &options{
		io:        f.IO(),
		apiClient: f.ApiClient,
		config:    f.Config(),
		baseRepo:  f.BaseRepo,
	}

	cmd := &cobra.Command{
		Use:     "set <key> <value>",
		Short:   "Create a new variable for a project or group.",
		Aliases: []string{"new", "create"},
		Args:    cobra.RangeArgs(1, 2),
		Example: heredoc.Doc(`
			$ glab variable set WITH_ARG "some value"
			$ glab variable set WITH_DESC "some value" --description "some description"
			$ glab variable set FROM_FLAG -v "some value"
			$ glab variable set FROM_ENV_WITH_ARG "${ENV_VAR}"
			$ glab variable set FROM_ENV_WITH_FLAG -v"${ENV_VAR}"
			$ glab variable set FROM_FILE < secret.txt
			$ cat file.txt | glab variable set SERVER_TOKEN
			$ cat token.txt | glab variable set GROUP_TOKEN -g mygroup --scope=prod
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.complete(args)

			if err := opts.validate(cmd, args); err != nil {
				return err
			}

			if runE != nil {
				return runE(opts)
			}
			return opts.run()
		},
	}

	cmd.Flags().StringVarP(&opts.value, "value", "v", "", "The value of a variable.")
	cmd.Flags().StringVarP(&opts.typ, "type", "t", "env_var", "The type of a variable: env_var, file.")
	cmd.Flags().StringVarP(&opts.scope, "scope", "s", "*", "The environment_scope of the variable. Values: all (*), or specific environments.")
	cmd.Flags().StringVarP(&opts.group, "group", "g", "", "Set variable for a group.")
	cmd.Flags().BoolVarP(&opts.masked, "masked", "m", false, "Whether the variable is masked.")
	cmd.Flags().BoolVarP(&opts.raw, "raw", "r", false, "Whether the variable is treated as a raw string.")
	cmd.Flags().BoolVarP(&opts.protected, "protected", "p", false, "Whether the variable is protected.")
	cmd.Flags().StringVarP(&opts.description, "description", "d", "", "Set description of a variable.")
	return cmd
}

func (o *options) complete(args []string) {
	o.key = args[0]
}

func (o *options) validate(cmd *cobra.Command, args []string) error {
	if !variableutils.IsValidKey(o.key) {
		return cmdutils.FlagError{Err: fmt.Errorf("invalid key provided.\n%s", variableutils.ValidKeyMsg)}
	}

	if o.value != "" && len(args) == 2 {
		return cmdutils.FlagError{Err: errors.New("specify value either by the second positional argument or the --value flag.")}
	}

	value, err := variableutils.GetValue(o.value, o.io, args)
	if err != nil {
		return err
	}
	o.value = value

	if cmd.Flags().Changed("type") {
		if o.typ != "env_var" && o.typ != "file" {
			return cmdutils.FlagError{Err: fmt.Errorf("invalid type: %s. --type must be one of `env_var` or `file`.", o.typ)}
		}
	}

	return nil
}

func (o *options) run() error {
	c := o.io.Color()

	// NOTE: this command can not only be used for projects,
	// so we have to manually check for the base repo, it it doesn't exist,
	// we bootstrap the client with the default hostname.
	var repoHost string
	if baseRepo, err := o.baseRepo(); err == nil {
		repoHost = baseRepo.RepoHost()
	}
	apiClient, err := o.apiClient(repoHost, o.config)
	if err != nil {
		return err
	}
	client := apiClient.Lab()

	if o.group != "" {
		// creating group-level variable
		createVarOpts := &gitlab.CreateGroupVariableOptions{
			Key:              gitlab.Ptr(o.key),
			Value:            gitlab.Ptr(o.value),
			EnvironmentScope: gitlab.Ptr(o.scope),
			Masked:           gitlab.Ptr(o.masked),
			Protected:        gitlab.Ptr(o.protected),
			VariableType:     gitlab.Ptr(gitlab.VariableTypeValue(o.typ)),
			Raw:              gitlab.Ptr(o.raw),
			Description:      gitlab.Ptr(o.description),
		}

		_, _, err := client.GroupVariables.CreateVariable(o.group, createVarOpts)
		if err != nil {
			return err
		}

		fmt.Fprintf(o.io.StdOut, "%s Created variable %s for group %s.\n", c.GreenCheck(), o.key, o.group)
		return nil
	}

	// creating project-level variable
	baseRepo, err := o.baseRepo()
	if err != nil {
		return err
	}
	createVarOpts := &gitlab.CreateProjectVariableOptions{
		Key:              gitlab.Ptr(o.key),
		Value:            gitlab.Ptr(o.value),
		EnvironmentScope: gitlab.Ptr(o.scope),
		Masked:           gitlab.Ptr(o.masked),
		Protected:        gitlab.Ptr(o.protected),
		VariableType:     gitlab.Ptr(gitlab.VariableTypeValue(o.typ)),
		Raw:              gitlab.Ptr(o.raw),
		Description:      gitlab.Ptr(o.description),
	}
	_, _, err = client.ProjectVariables.CreateVariable(baseRepo.FullName(), createVarOpts)
	if err != nil {
		return err
	}

	fmt.Fprintf(o.io.StdOut, "%s Created variable %s for %s with scope %s.\n", c.GreenCheck(), o.key, baseRepo.FullName(), o.scope)
	return nil
}
