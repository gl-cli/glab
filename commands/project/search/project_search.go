package search

import (
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"gitlab.com/gitlab-org/cli/pkg/tableprinter"

	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/pkg/utils"

	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

func NewCmdSearch(f *cmdutils.Factory) *cobra.Command {
	projectSearchCmd := &cobra.Command{
		Use:     "search [flags]",
		Short:   `Search for GitLab repositories and projects by name.`,
		Long:    ``,
		Args:    cobra.ExactArgs(0),
		Aliases: []string{"find", "lookup"},
		Example: heredoc.Doc(`
			- glab project search -s "title"
			- glab repo search -s "title"
			- glab project find -s "title"
			- glab project lookup -s "title"
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			c := f.IO.Color()
			apiClient, err := f.HttpClient()
			if err != nil {
				return err
			}

			search, _ := cmd.Flags().GetString("search")
			page, _ := cmd.Flags().GetInt("page")
			perPage, _ := cmd.Flags().GetInt("per-page")

			listOpts := gitlab.ListOptions{
				Page:    page,
				PerPage: perPage,
			}
			projects, _, err := apiClient.Search.Projects(search, &gitlab.SearchOptions{ListOptions: listOpts})
			if err != nil {
				return err
			}

			title := fmt.Sprintf("Showing results for \"%s\"", search)
			if len(projects) == 0 {
				title = fmt.Sprintf("No results found for \"%s\"", search)
			}

			table := tableprinter.NewTablePrinter()
			table.Wrap = true
			for _, p := range projects {
				table.AddCell(c.Green(string(rune(p.ID))))

				var description string
				if p.Description != "" {
					description = fmt.Sprintf("\n%s", c.Cyan(p.Description))
				}

				table.AddCellf("%s%s\n%s", p.PathWithNamespace, description, c.Gray(p.WebURL))
				table.AddCellf("%d stars %d forks %d issues", p.StarCount, p.ForksCount, p.OpenIssuesCount)
				table.AddCellf("updated %s", utils.TimeToPrettyTimeAgo(*p.LastActivityAt))
				table.EndRow()
			}

			fmt.Fprintf(f.IO.StdOut, "%s\n%s\n", title, table.Render())
			return nil
		},
	}

	projectSearchCmd.Flags().IntP("page", "p", 1, "Page number.")
	projectSearchCmd.Flags().IntP("per-page", "P", 20, "Number of items to list per page.")
	projectSearchCmd.Flags().StringP("search", "s", "", "A string contained in the project name.")
	_ = projectSearchCmd.MarkFlagRequired("Search.")

	return projectSearchCmd
}
