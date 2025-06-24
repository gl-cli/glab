package list

import (
	"encoding/json"
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"

	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/commands/flag"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/pkg/iostreams"
	"gitlab.com/gitlab-org/cli/pkg/tableprinter"
)

type options struct {
	httpClient func() (*gitlab.Client, error)
	io         *iostreams.IOStreams
	baseRepo   func() (glrepo.Interface, error)
	page       int
	perPage    int

	valueSet     bool
	group        string
	outputFormat string
}

func NewCmdList(f cmdutils.Factory, runE func(opts *options) error) *cobra.Command {
	opts := &options{
		io:         f.IO(),
		httpClient: f.HttpClient,
		baseRepo:   f.BaseRepo,
	}

	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List variables for a project or group.",
		Aliases: []string{"ls"},
		Args:    cobra.ExactArgs(0),
		Example: heredoc.Doc(`
			$ glab variable list
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

	return cmd
}

func (o *options) complete(cmd *cobra.Command) error {
	group, err := flag.GroupOverride(cmd)
	if err != nil {
		return err
	}
	o.group = group

	return nil
}

func (o *options) run() error {
	color := o.io.Color()
	httpClient, err := o.httpClient()
	if err != nil {
		return err
	}

	table := tableprinter.NewTablePrinter()
	table.AddRow("KEY", "PROTECTED", "MASKED", "EXPANDED", "SCOPE", "DESCRIPTION")

	if o.group != "" {
		o.io.Logf("Listing variables for the %s group:\n\n", color.Bold(o.group))
		createVarOpts := &gitlab.ListGroupVariablesOptions{Page: o.page, PerPage: o.perPage}
		variables, err := api.ListGroupVariables(httpClient, o.group, createVarOpts)
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
	} else {
		repo, err := o.baseRepo()
		if err != nil {
			return err
		}
		o.io.Logf("Listing variables for the %s project:\n\n", color.Bold(repo.FullName()))
		createVarOpts := &gitlab.ListProjectVariablesOptions{Page: o.page, PerPage: o.perPage}
		variables, err := api.ListProjectVariables(httpClient, repo.FullName(), createVarOpts)
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
