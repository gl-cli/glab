package contributors

import (
	"fmt"

	"gitlab.com/gitlab-org/cli/internal/iostreams"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/internal/tableprinter"
	"gitlab.com/gitlab-org/cli/internal/utils"
)

type options struct {
	orderBy string
	sort    string
	perPage int
	page    int

	baseRepo   func() (glrepo.Interface, error)
	httpClient func() (*gitlab.Client, error)
	io         *iostreams.IOStreams
}

func NewCmdContributors(f cmdutils.Factory) *cobra.Command {
	opts := &options{
		io:         f.IO(),
		baseRepo:   f.BaseRepo,
		httpClient: f.HttpClient,
	}
	repoContributorsCmd := &cobra.Command{
		Use:   "contributors",
		Short: `Get repository contributors list.`,
		Example: heredoc.Doc(`
			# List contributors for the current repository
			$ glab repo contributors

			# List contributors for a specific repository
			$ glab repo contributors -R gitlab-com/www-gitlab-com
		`),
		Args:    cobra.ExactArgs(0),
		Aliases: []string{"users"},
		RunE: func(cmd *cobra.Command, args []string) error {
			return opts.run()
		},
	}

	cmdutils.EnableRepoOverride(repoContributorsCmd, f)

	repoContributorsCmd.Flags().StringVarP(&opts.orderBy, "order", "o", "commits", "Return contributors ordered by name, email, or commits (orders by commit date) fields.")
	repoContributorsCmd.Flags().StringVarP(&opts.sort, "sort", "s", "", "Return contributors. Sort options: asc, desc.")
	repoContributorsCmd.Flags().IntVarP(&opts.page, "page", "p", 1, "Page number.")
	repoContributorsCmd.Flags().IntVarP(&opts.perPage, "per-page", "P", 30, "Number of items to list per page.")
	return repoContributorsCmd
}

func (o *options) run() error {
	var err error
	c := o.io.Color()

	apiClient, err := o.httpClient()
	if err != nil {
		return err
	}

	repo, err := o.baseRepo()
	if err != nil {
		return err
	}

	if o.orderBy == "commits" && o.sort == "" {
		o.sort = "desc"
	}

	l := &gitlab.ListContributorsOptions{
		OrderBy: gitlab.Ptr(o.orderBy),
		ListOptions: gitlab.ListOptions{
			Page:    o.page,
			PerPage: o.perPage,
		},
	}

	if o.sort != "" {
		l.Sort = gitlab.Ptr(o.sort)
	}

	users, _, err := apiClient.Repositories.Contributors(repo.FullName(), l)
	if err != nil {
		return err
	}

	// Title
	title := utils.NewListTitle("contributor")
	title.RepoName = repo.FullName()
	title.Page = l.Page
	title.CurrentPageTotal = len(users)

	// List
	table := tableprinter.NewTablePrinter()
	for _, user := range users {
		table.AddCell(user.Name)
		table.AddCellf("%s", c.Gray(user.Email))
		table.AddCellf("%d commits", user.Commits)
		table.EndRow()
	}

	fmt.Fprintf(o.io.StdOut, "%s\n%s\n", title.Describe(), table.String())
	return err
}
