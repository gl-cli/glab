package delete

import (
	"errors"
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/cli/internal/api"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/commands/variable/variableutils"
	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/internal/iostreams"

	"github.com/spf13/cobra"
)

type options struct {
	apiClient func(repoHost string, cfg config.Config) (*api.Client, error)
	config    config.Config
	io        *iostreams.IOStreams
	baseRepo  func() (glrepo.Interface, error)

	key   string
	scope string
	group string
}

func NewCmdDelete(f cmdutils.Factory) *cobra.Command {
	opts := &options{
		io:        f.IO(),
		apiClient: f.ApiClient,
		config:    f.Config(),
		baseRepo:  f.BaseRepo,
	}

	cmd := &cobra.Command{
		Use:     "delete <key>",
		Short:   "Delete a variable for a project or group.",
		Aliases: []string{"remove"},
		Args:    cobra.ExactArgs(1),
		Example: heredoc.Doc(`
	    - glab variable delete VAR_NAME
		  - glab variable delete VAR_NAME --scope=prod
		  - glab variable delete VARNAME -g mygroup
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.complete(args)

			if err := opts.validate(cmd); err != nil {
				return err
			}

			return opts.run()
		},
	}

	cmd.Flags().StringVarP(&opts.scope, "scope", "s", "*", "The 'environment_scope' of the variable. Options: all (*), or specific environments.")
	cmd.Flags().StringVarP(&opts.group, "group", "g", "", "Delete variable from a group.")

	return cmd
}

func (o *options) complete(args []string) {
	o.key = args[0]
}

func (o *options) validate(cmd *cobra.Command) error {
	if !variableutils.IsValidKey(o.key) {
		return cmdutils.FlagError{Err: fmt.Errorf("invalid key provided.\n%s", variableutils.ValidKeyMsg)}
	}

	if cmd.Flags().Changed("scope") && o.group != "" {
		return cmdutils.FlagError{Err: errors.New("scope is not required for group variables.")}
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

	if o.group == "" {
		// Delete project-level variable
		baseRepo, err := o.baseRepo()
		if err != nil {
			return err
		}

		_, err = client.ProjectVariables.RemoveVariable(baseRepo.FullName(), o.key, &gitlab.RemoveProjectVariableOptions{
			Filter: &gitlab.VariableFilter{EnvironmentScope: o.scope},
		})
		if err != nil {
			return err
		}

		fmt.Fprintf(o.io.StdOut, "%s Deleted variable %s with scope %s for %s.\n", c.GreenCheck(), o.key, o.scope, baseRepo.FullName())
	} else {
		// Delete group-level variable
		_, err := client.GroupVariables.RemoveVariable(o.group, o.key, nil)
		if err != nil {
			return err
		}

		fmt.Fprintf(o.io.StdOut, "%s Deleted variable %s for group %s.\n", c.GreenCheck(), o.key, o.group)
	}

	return nil
}
