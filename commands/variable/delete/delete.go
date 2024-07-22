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
	"github.com/xanzy/go-gitlab"
)

type DeleteOpts struct {
	HTTPClient func() (*gitlab.Client, error)
	IO         *iostreams.IOStreams
	BaseRepo   func() (glrepo.Interface, error)

	Key   string
	Scope string
	Group string
}

func NewCmdSet(f *cmdutils.Factory, runE func(opts *DeleteOpts) error) *cobra.Command {
	opts := &DeleteOpts{
		IO: f.IO,
	}

	cmd := &cobra.Command{
		Use:     "delete <key>",
		Short:   "Delete a variable for a project or group.",
		Aliases: []string{"remove"},
		Args:    cobra.ExactArgs(1),
		Example: heredoc.Doc(`
			glab variable delete VAR_NAME
			glab variable delete VAR_NAME --scope=prod
			glab variable delete VARNAME -g mygroup
		`),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			opts.HTTPClient = f.HttpClient
			opts.BaseRepo = f.BaseRepo
			opts.Key = args[0]

			if !variableutils.IsValidKey(opts.Key) {
				err = cmdutils.FlagError{Err: fmt.Errorf("invalid key provided.\n%s", variableutils.ValidKeyMsg)}
				return err
			} else if len(args) != 1 {
				err = cmdutils.FlagError{Err: errors.New("no key provided.")}
				return err
			}

			if cmd.Flags().Changed("scope") && opts.Group != "" {
				err = cmdutils.FlagError{Err: errors.New("scope is not required for group variables.")}
				return err
			}

			if runE != nil {
				err = runE(opts)
				return err
			}
			err = deleteRun(opts)
			return err
		},
	}

	cmd.Flags().StringVarP(&opts.Scope, "scope", "s", "*", "The 'environment_scope' of the variable. Options: all (*), or specific environments.")
	cmd.Flags().StringVarP(&opts.Group, "group", "g", "", "Delete variable from a group.")

	return cmd
}

func deleteRun(opts *DeleteOpts) error {
	c := opts.IO.Color()
	httpClient, err := opts.HTTPClient()
	if err != nil {
		return err
	}

	baseRepo, err := opts.BaseRepo()
	if err != nil {
		return err
	}

	if opts.Group == "" {
		// Delete project-level variable
		err = api.DeleteProjectVariable(httpClient, baseRepo.FullName(), opts.Key, opts.Scope)
		if err != nil {
			return err
		}

		fmt.Fprintf(opts.IO.StdOut, "%s Deleted variable %s with scope %s for %s.\n", c.GreenCheck(), opts.Key, opts.Scope, baseRepo.FullName())
	} else {
		// Delete group-level variable
		err = api.DeleteGroupVariable(httpClient, opts.Group, opts.Key)
		if err != nil {
			return err
		}

		fmt.Fprintf(opts.IO.StdOut, "%s Deleted variable %s for group %s.\n", c.GreenCheck(), opts.Key, opts.Group)
	}

	return nil
}
