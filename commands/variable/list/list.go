package list

import (
	"encoding/json"
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	"github.com/xanzy/go-gitlab"

	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/commands/flag"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/pkg/iostreams"
	"gitlab.com/gitlab-org/cli/pkg/tableprinter"
)

type ListOpts struct {
	HTTPClient func() (*gitlab.Client, error)
	IO         *iostreams.IOStreams
	BaseRepo   func() (glrepo.Interface, error)

	ValueSet     bool
	Group        string
	OutputFormat string
}

func NewCmdSet(f *cmdutils.Factory, runE func(opts *ListOpts) error) *cobra.Command {
	opts := &ListOpts{
		IO: f.IO,
	}

	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List variables for a project or group.",
		Aliases: []string{"ls"},
		Args:    cobra.ExactArgs(0),
		Example: heredoc.Doc(
			`
			glab variable list
		`,
		),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			// Supports repo override
			opts.HTTPClient = f.HttpClient
			opts.BaseRepo = f.BaseRepo

			group, err := flag.GroupOverride(cmd)
			if err != nil {
				return err
			}
			opts.Group = group

			if runE != nil {
				err = runE(opts)
				return
			}
			err = listRun(opts)
			return
		},
	}

	cmdutils.EnableRepoOverride(cmd, f)
	cmd.PersistentFlags().StringP(
		"group",
		"g",
		"",
		"Select a group or subgroup. Ignored if a repository argument is set.",
	)
	cmd.Flags().StringVarP(&opts.OutputFormat, "output", "F", "text", "Format output as: text, json.")

	return cmd
}

func listRun(opts *ListOpts) error {
	color := opts.IO.Color()
	httpClient, err := opts.HTTPClient()
	if err != nil {
		return err
	}

	repo, err := opts.BaseRepo()
	if err != nil {
		return err
	}

	table := tableprinter.NewTablePrinter()
	table.AddRow("KEY", "PROTECTED", "MASKED", "EXPANDED", "SCOPE")

	if opts.Group != "" {
		opts.IO.Logf("Listing variables for the %s group:\n\n", color.Bold(opts.Group))
		createVarOpts := &gitlab.ListGroupVariablesOptions{}
		variables, err := api.ListGroupVariables(httpClient, opts.Group, createVarOpts)
		if err != nil {
			return err
		}
		if opts.OutputFormat == "json" {
			varListJSON, _ := json.Marshal(variables)
			fmt.Fprintln(opts.IO.StdOut, string(varListJSON))

		} else {
			for _, variable := range variables {
				table.AddRow(variable.Key, variable.Protected, variable.Masked, !variable.Raw, variable.EnvironmentScope)
			}
		}
	} else {
		opts.IO.Logf("Listing variables for the %s project:\n\n", color.Bold(repo.FullName()))
		createVarOpts := &gitlab.ListProjectVariablesOptions{}
		variables, err := api.ListProjectVariables(httpClient, repo.FullName(), createVarOpts)
		if err != nil {
			return err
		}
		if opts.OutputFormat == "json" {
			varListJSON, _ := json.Marshal(variables)
			fmt.Fprintln(opts.IO.StdOut, string(varListJSON))
		} else {
			for _, variable := range variables {
				table.AddRow(variable.Key, variable.Protected, variable.Masked, !variable.Raw, variable.EnvironmentScope)
			}
		}
	}

	if opts.OutputFormat != "json" {
		opts.IO.Log(table.String())
	}
	return nil
}
