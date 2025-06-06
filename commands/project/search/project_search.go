package search

import (
	"encoding/json"
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"gitlab.com/gitlab-org/cli/pkg/iostreams"
	"gitlab.com/gitlab-org/cli/pkg/tableprinter"

	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/pkg/utils"

	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

type Options struct {
	PerPage      int
	Page         int
	Search       string
	OutputFormat string
	HTTPClient   func() (*gitlab.Client, error)
	IO           *iostreams.IOStreams
}

func NewCmdSearch(f cmdutils.Factory) *cobra.Command {
	opts := &Options{
		IO: f.IO,
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
			opts.HTTPClient = f.HttpClient
			apiClient, err := opts.HTTPClient()
			if err != nil {
				return err
			}
			search := opts.Search
			page := opts.Page
			perPage := opts.PerPage

			listOpts := gitlab.ListOptions{
				Page:    page,
				PerPage: perPage,
			}
			projects, _, err := apiClient.Search.Projects(search, &gitlab.SearchOptions{ListOptions: listOpts})
			if err != nil {
				return err
			}
			if opts.OutputFormat == "json" {
				projectListJSON, err := json.Marshal(projects)
				if err != nil {
					return err
				}
				fmt.Fprintln(opts.IO.StdOut, string(projectListJSON))
				return nil
			}
			title := fmt.Sprintf("Showing results for \"%s\"", search)
			if len(projects) == 0 {
				title = fmt.Sprintf("No results found for \"%s\"", search)
			}
			table := tableprinter.NewTablePrinter()
			table.Wrap = true
			for _, p := range projects {
				table.AddCell(opts.IO.Color().Green(fmt.Sprintf("%d", p.ID)))

				var description string
				if p.Description != "" {
					description = fmt.Sprintf("\n%s", opts.IO.Color().Cyan(p.Description))
				}

				table.AddCellf("%s%s\n%s", p.PathWithNamespace, description, opts.IO.Color().Gray(p.WebURL))
				table.AddCellf("%d stars %d forks %d issues", p.StarCount, p.ForksCount, p.OpenIssuesCount)
				table.AddCellf("updated %s", utils.TimeToPrettyTimeAgo(*p.LastActivityAt))
				table.EndRow()
			}

			fmt.Fprintf(f.IO.StdOut, "%s\n%s\n", title, table.Render())
			return nil
		},
	}

	projectSearchCmd.Flags().IntVarP(&opts.Page, "page", "p", 1, "Page number.")
	projectSearchCmd.Flags().IntVarP(&opts.PerPage, "per-page", "P", 20, "Number of items to list per page.")
	projectSearchCmd.Flags().StringVarP(&opts.Search, "search", "s", "", "A string contained in the project name.")
	projectSearchCmd.Flags().StringVarP(&opts.OutputFormat, "output", "F", "text", "Format output as: text, json.")
	_ = projectSearchCmd.MarkFlagRequired("Search.")

	return projectSearchCmd
}
