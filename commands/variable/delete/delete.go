package delete

import (
	"errors"
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/commands/variable/variableutils"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/pkg/iostreams"

	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

type options struct {
	httpClient func() (*gitlab.Client, error)
	io         *iostreams.IOStreams
	baseRepo   func() (glrepo.Interface, error)

	key   string
	scope string
	group string
}

func NewCmdDelete(f cmdutils.Factory) *cobra.Command {
	opts := &options{
		io:         f.IO(),
		httpClient: f.HttpClient,
		baseRepo:   f.BaseRepo,
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

			if err := opts.validate(cmd, args); err != nil {
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

func (o *options) validate(cmd *cobra.Command, args []string) error {
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
	httpClient, err := o.httpClient()
	if err != nil {
		return err
	}

	baseRepo, err := o.baseRepo()
	if err != nil {
		return err
	}

	if o.group == "" {
		// Delete project-level variable
		err = api.DeleteProjectVariable(httpClient, baseRepo.FullName(), o.key, o.scope)
		if err != nil {
			return err
		}

		fmt.Fprintf(o.io.StdOut, "%s Deleted variable %s with scope %s for %s.\n", c.GreenCheck(), o.key, o.scope, baseRepo.FullName())
	} else {
		// Delete group-level variable
		err = api.DeleteGroupVariable(httpClient, o.group, o.key)
		if err != nil {
			return err
		}

		fmt.Fprintf(o.io.StdOut, "%s Deleted variable %s for group %s.\n", c.GreenCheck(), o.key, o.group)
	}

	return nil
}
