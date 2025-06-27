package search

import (
	"encoding/json"
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"gitlab.com/gitlab-org/cli/internal/api"
	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/internal/iostreams"
	"gitlab.com/gitlab-org/cli/internal/tableprinter"

	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/utils"

	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

type options struct {
	perPage      int
	page         int
	search       string
	outputFormat string
	apiClient    func(repoHost string, cfg config.Config) (*api.Client, error)
	config       config.Config
	io           *iostreams.IOStreams
}

func NewCmdSearch(f cmdutils.Factory) *cobra.Command {
	opts := &options{
		io:        f.IO(),
		apiClient: f.ApiClient,
		config:    f.Config(),
	}

	projectSearchCmd := &cobra.Command{
		Use:     "search [flags]",
		Short:   `Search for GitLab repositories and projects by name.`,
		Long:    ``,
		Args:    cobra.ExactArgs(0),
		Aliases: []string{"find", "lookup"},
		Example: heredoc.Doc(`
			$ glab project search -s "title"
			$ glab repo search -s "title"
			$ glab project find -s "title"
			$ glab project lookup -s "title"
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			return opts.run()
		},
	}

	projectSearchCmd.Flags().IntVarP(&opts.page, "page", "p", 1, "Page number.")
	projectSearchCmd.Flags().IntVarP(&opts.perPage, "per-page", "P", 20, "Number of items to list per page.")
	projectSearchCmd.Flags().StringVarP(&opts.search, "search", "s", "", "A string contained in the project name.")
	projectSearchCmd.Flags().StringVarP(&opts.outputFormat, "output", "F", "text", "Format output as: text, json.")
	cobra.CheckErr(projectSearchCmd.MarkFlagRequired("search"))

	return projectSearchCmd
}

func (o *options) run() error {
	c, err := o.apiClient("", o.config)
	if err != nil {
		return err
	}
	gitlabClient := c.Lab()
	listOpts := gitlab.ListOptions{
		Page:    o.page,
		PerPage: o.perPage,
	}
	projects, _, err := gitlabClient.Search.Projects(o.search, &gitlab.SearchOptions{ListOptions: listOpts})
	if err != nil {
		return err
	}
	if o.outputFormat == "json" {
		projectListJSON, err := json.Marshal(projects)
		if err != nil {
			return err
		}
		fmt.Fprintln(o.io.StdOut, string(projectListJSON))
		return nil
	}
	title := fmt.Sprintf("Showing results for \"%s\"", o.search)
	if len(projects) == 0 {
		title = fmt.Sprintf("No results found for \"%s\"", o.search)
	}
	table := tableprinter.NewTablePrinter()
	table.Wrap = true
	for _, p := range projects {
		table.AddCell(o.io.Color().Green(fmt.Sprintf("%d", p.ID)))

		var description string
		if p.Description != "" {
			description = fmt.Sprintf("\n%s", o.io.Color().Cyan(p.Description))
		}

		table.AddCellf("%s%s\n%s", p.PathWithNamespace, description, o.io.Color().Gray(p.WebURL))
		table.AddCellf("%d stars %d forks %d issues", p.StarCount, p.ForksCount, p.OpenIssuesCount)
		table.AddCellf("updated %s", utils.TimeToPrettyTimeAgo(*p.LastActivityAt))
		table.EndRow()
	}

	fmt.Fprintf(o.io.StdOut, "%s\n%s\n", title, table.Render())
	return nil
}
