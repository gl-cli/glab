package export

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
)

type ExportOpts struct {
	HTTPClient func() (*gitlab.Client, error)
	IO         *iostreams.IOStreams
	BaseRepo   func() (glrepo.Interface, error)

	ValueSet bool
	Group    string

	Page    int
	PerPage int
}

func marshalJson(variables interface{}) ([]byte, error) {
	res, err := json.MarshalIndent(variables, "", "  ")
	if err != nil {
		return nil, err
	}

	return res, nil
}

func NewCmdExport(f *cmdutils.Factory, runE func(opts *ExportOpts) error) *cobra.Command {
	opts := &ExportOpts{
		IO: f.IO,
	}

	cmd := &cobra.Command{
		Use:     "export",
		Short:   "Export variables from a project or group.",
		Aliases: []string{"ex"},
		Args:    cobra.ExactArgs(0),
		Example: heredoc.Doc(`
			glab variable export
			glab variable export --per-page 1000 --page 1
			glab variable export --group gitlab-org
			glab variable export --group gitlab-org --per-page 1000 --page 1
		`),
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
			err = exportRun(opts)
			return
		},
	}

	cmdutils.EnableRepoOverride(cmd, f)
	cmd.PersistentFlags().StringP("group", "g", "", "Select a group or subgroup. Ignored if a repository argument is set.")
	cmd.Flags().IntVarP(&opts.Page, "page", "p", 1, "Page number.")
	cmd.Flags().IntVarP(&opts.PerPage, "per-page", "P", 100, "Number of items to list per page.")
	return cmd
}

func exportRun(opts *ExportOpts) error {
	httpClient, err := opts.HTTPClient()
	if err != nil {
		return err
	}

	repo, err := opts.BaseRepo()
	if err != nil {
		return err
	}

	if opts.Group != "" {

		createVarOpts := &gitlab.ListGroupVariablesOptions{Page: opts.Page, PerPage: opts.PerPage}
		groupVariables, err := api.ListGroupVariables(httpClient, opts.Group, createVarOpts)
		if err != nil {
			return err
		}

		opts.IO.Logf("Exporting variables from the %s group:\n", opts.Group)

		if len(groupVariables) == 0 {
			return nil
		}

		res, err := marshalJson(groupVariables)
		if err != nil {
			return err
		}

		fmt.Println(string(res))

	} else {

		createVarOpts := &gitlab.ListProjectVariablesOptions{Page: opts.Page, PerPage: opts.PerPage}
		projectVariables, err := api.ListProjectVariables(httpClient, repo.FullName(), createVarOpts)
		if err != nil {
			return err
		}

		opts.IO.Logf("Exporting variables from the %s project:\n", repo.FullName())

		if len(projectVariables) == 0 {
			return nil
		}

		res, err := marshalJson(projectVariables)
		if err != nil {
			return err
		}

		fmt.Println(string(res))

	}

	return nil
}
