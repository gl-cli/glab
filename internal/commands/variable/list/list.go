package list

import (
	"encoding/json"
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"

	"gitlab.com/gitlab-org/cli/internal/api"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/internal/iostreams"
	"gitlab.com/gitlab-org/cli/internal/tableprinter"
)

type options struct {
	apiClient func(repoHost string, cfg config.Config) (*api.Client, error)
	config    config.Config
	io        *iostreams.IOStreams
	baseRepo  func() (glrepo.Interface, error)
	page      int
	perPage   int

	group        string
	outputFormat string
	instance     bool
}

func NewCmdList(f cmdutils.Factory, runE func(opts *options) error) *cobra.Command {
	opts := &options{
		io:        f.IO(),
		apiClient: f.ApiClient,
		config:    f.Config(),
		baseRepo:  f.BaseRepo,
	}

	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List variables for a project or group.",
		Aliases: []string{"ls"},
		Args:    cobra.ExactArgs(0),
		Example: heredoc.Doc(`
			$ glab variable list
			$ glab variable list -i
			$ glab variable list --per-page 100 --page 1
			$ glab variable list --group gitlab-org
			$ glab variable list --group gitlab-org --per-page 100
		`,
		),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := opts.complete(cmd); err != nil {
				return err
			}

			if runE != nil {
				return runE(opts)
			}

			return opts.run()
		},
	}

	cmdutils.EnableRepoOverride(cmd, f)
	cmd.PersistentFlags().StringP("group", "g", "", "Select a group or subgroup. Ignored if a repository argument is set.")
	cmd.Flags().StringVarP(&opts.outputFormat, "output", "F", "text", "Format output as: text, json.")
	cmd.Flags().IntVarP(&opts.perPage, "per-page", "P", 20, "Number of items to list per page.")
	cmd.Flags().IntVarP(&opts.page, "page", "p", 1, "Page number.")
	cmd.Flags().BoolVarP(&opts.instance, "instance", "i", false, "Display instance variables.")

	return cmd
}

func (o *options) complete(cmd *cobra.Command) error {
	group, err := cmdutils.GroupOverride(cmd)
	if err != nil {
		return err
	}
	o.group = group

	return nil
}

func (o *options) run() error {
	color := o.io.Color()

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

	table := tableprinter.NewTablePrinter()
	table.AddRow("KEY", "PROTECTED", "MASKED", "EXPANDED", "SCOPE", "DESCRIPTION")

	if o.group != "" {
		o.io.Logf("Listing variables for the %s group:\n\n", color.Bold(o.group))
		listOpts := &gitlab.ListGroupVariablesOptions{Page: o.page, PerPage: o.perPage}
		variables, _, err := client.GroupVariables.ListVariables(o.group, listOpts)
		if err != nil {
			return err
		}
		if o.outputFormat == "json" {
			varListJSON, _ := json.Marshal(variables)
			fmt.Fprintln(o.io.StdOut, string(varListJSON))

		} else {
			for _, variable := range variables {
				table.AddRow(variable.Key, variable.Protected, variable.Masked, !variable.Raw, variable.EnvironmentScope, variable.Description)
			}
		}
	} else if o.instance {
		o.io.Logf("Listing variables for the instance\n\n")
		listOpts := &gitlab.ListInstanceVariablesOptions{Page: o.page, PerPage: o.perPage}
		variables, _, err := client.InstanceVariables.ListVariables(listOpts)
		if err != nil {
			return err
		}
		if o.outputFormat == "json" {
			varListJSON, _ := json.Marshal(variables)
			fmt.Fprintln(o.io.StdOut, string(varListJSON))

		} else {
			for _, variable := range variables {
				table.AddRow(variable.Key, variable.Protected, variable.Masked, !variable.Raw, "", variable.Description)
			}
		}
	} else {
		repo, err := o.baseRepo()
		if err != nil {
			return err
		}
		o.io.Logf("Listing variables for the %s project:\n\n", color.Bold(repo.FullName()))
		listOpts := &gitlab.ListProjectVariablesOptions{Page: o.page, PerPage: o.perPage}
		variables, _, err := client.ProjectVariables.ListVariables(repo.FullName(), listOpts)
		if err != nil {
			return err
		}
		if o.outputFormat == "json" {
			varListJSON, _ := json.Marshal(variables)
			fmt.Fprintln(o.io.StdOut, string(varListJSON))
		} else {
			for _, variable := range variables {
				table.AddRow(variable.Key, variable.Protected, variable.Masked, !variable.Raw, variable.EnvironmentScope, variable.Description)
			}
		}
	}

	if o.outputFormat != "json" {
		fmt.Fprint(o.io.StdOut, table.String())
	}
	return nil
}
