package list

import (
	"encoding/json"
	"fmt"

	"gitlab.com/gitlab-org/cli/pkg/iostreams"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	"github.com/xanzy/go-gitlab"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/pkg/tableprinter"
)

type Options struct {
	OrderBy          string
	Sort             string
	Group            string
	IncludeSubgroups bool
	PerPage          int
	Page             int
	OutputFormat     string
	FilterAll        bool
	FilterOwned      bool
	FilterMember     bool
	FilterStarred    bool
	Archived         bool
	ArchivedSet      bool

	HTTPClient func() (*gitlab.Client, error)
	IO         *iostreams.IOStreams
}

func NewCmdList(f *cmdutils.Factory) *cobra.Command {
	opts := &Options{
		IO: f.IO,
	}
	repoListCmd := &cobra.Command{
		Use:   "list",
		Short: `Get list of repositories.`,
		Example: heredoc.Doc(`
	glab repo list
	`),
		Args:    cobra.ExactArgs(0),
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.HTTPClient = f.HttpClient
			opts.ArchivedSet = cmd.Flags().Changed("archived")

			return runE(opts)
		},
	}

	repoListCmd.Flags().StringVarP(&opts.OrderBy, "order", "o", "last_activity_at", "Return repositories ordered by id, created_at, or other fields.")
	repoListCmd.Flags().StringVarP(&opts.Sort, "sort", "s", "", "Return repositories sorted in asc or desc order.")
	repoListCmd.Flags().StringVarP(&opts.Group, "group", "g", "", "Return repositories in only the given group.")
	repoListCmd.Flags().BoolVarP(&opts.IncludeSubgroups, "include-subgroups", "G", false, "Include projects in subgroups of this group. Default is false. Used with the '--group' flag.")
	repoListCmd.Flags().IntVarP(&opts.Page, "page", "p", 1, "Page number.")
	repoListCmd.Flags().IntVarP(&opts.PerPage, "per-page", "P", 30, "Number of items to list per page.")
	repoListCmd.Flags().StringVarP(&opts.OutputFormat, "output", "F", "text", "Format output as: text, json.")
	repoListCmd.Flags().BoolVarP(&opts.FilterAll, "all", "a", false, "List all projects on the instance.")
	repoListCmd.Flags().BoolVarP(&opts.FilterOwned, "mine", "m", false, "List only projects you own. Default if no filters are provided.")
	repoListCmd.Flags().BoolVar(&opts.FilterMember, "member", false, "List only projects of which you are a member.")
	repoListCmd.Flags().BoolVar(&opts.FilterStarred, "starred", false, "List only starred projects.")
	repoListCmd.Flags().BoolVar(&opts.Archived, "archived", false, "Limit by archived status. Used with the '--group' flag.")
	return repoListCmd
}

func runE(opts *Options) error {
	var err error
	c := opts.IO.Color()

	apiClient, err := opts.HTTPClient()
	if err != nil {
		return err
	}

	var projects []*gitlab.Project
	var resp *gitlab.Response
	if len(opts.Group) > 0 {
		projects, resp, err = listAllProjectsForGroup(apiClient, *opts)
	} else {
		projects, resp, err = listAllProjects(apiClient, *opts)
	}

	if err != nil {
		return err
	}

	if opts.OutputFormat == "json" {
		projectListJSON, _ := json.Marshal(projects)
		fmt.Fprintln(opts.IO.StdOut, string(projectListJSON))
	} else {
		// Title
		title := fmt.Sprintf("Showing %d of %d projects (Page %d of %d).\n", len(projects), resp.TotalItems, resp.CurrentPage, resp.TotalPages)

		// List
		table := tableprinter.NewTablePrinter()
		for _, prj := range projects {
			table.AddCell(c.Blue(prj.PathWithNamespace))
			table.AddCell(prj.SSHURLToRepo)
			table.AddCell(prj.Description)
			table.EndRow()
		}

		fmt.Fprintf(opts.IO.StdOut, "%s\n%s\n", title, table.String())
	}

	return err
}

func listAllProjects(apiClient *gitlab.Client, opts Options) ([]*gitlab.Project, *gitlab.Response, error) {
	l := &gitlab.ListProjectsOptions{
		OrderBy: gitlab.Ptr(opts.OrderBy),
	}

	// Other filters only valid if FilterAll not true
	if !opts.FilterAll {
		if !opts.FilterStarred && !opts.FilterMember {
			// if no other filters are passed, default to Owned filter
			l.Owned = gitlab.Ptr(true)
		}

		if opts.FilterOwned {
			l.Owned = gitlab.Ptr(opts.FilterOwned)
		}

		if opts.FilterStarred {
			l.Starred = gitlab.Ptr(opts.FilterStarred)
		}

		if opts.FilterMember {
			l.Membership = gitlab.Ptr(opts.FilterMember)
		}
	}

	if opts.Sort != "" {
		l.Sort = gitlab.Ptr(opts.Sort)
	}

	l.PerPage = opts.PerPage
	l.Page = opts.Page

	return apiClient.Projects.ListProjects(l)
}

func listAllProjectsForGroup(apiClient *gitlab.Client, opts Options) ([]*gitlab.Project, *gitlab.Response, error) {
	groups, resp, err := apiClient.Groups.SearchGroup(opts.Group)
	if err != nil {
		return nil, resp, err
	}

	var group *gitlab.Group = nil
	for _, g := range groups {
		if g.FullPath == opts.Group {
			group = g
			break
		}
	}
	if group == nil {
		return nil, nil, fmt.Errorf("No group matching path %s", opts.Group)
	}

	l := &gitlab.ListGroupProjectsOptions{
		OrderBy: gitlab.Ptr(opts.OrderBy),
	}

	// Other filters only valid if FilterAll not true
	if !opts.FilterAll {
		if !opts.FilterStarred && !opts.FilterMember {
			// if no other filters are passed, default to Owned filter
			l.Owned = gitlab.Ptr(true)
		}

		if opts.FilterOwned {
			l.Owned = gitlab.Ptr(opts.FilterOwned)
		}

		if opts.FilterStarred {
			l.Starred = gitlab.Ptr(opts.FilterStarred)
		}

		if opts.IncludeSubgroups {
			l.IncludeSubGroups = gitlab.Ptr(true)
		}

		if opts.ArchivedSet {
			l.Archived = gitlab.Ptr(opts.Archived)
		}
	}

	if opts.Sort != "" {
		l.Sort = gitlab.Ptr(opts.Sort)
	}

	l.PerPage = opts.PerPage
	l.Page = opts.Page

	return apiClient.Groups.ListGroupProjects(group.ID, l)
}
