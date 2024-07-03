package contributors

import (
	"fmt"

	"gitlab.com/gitlab-org/cli/pkg/iostreams"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	"github.com/xanzy/go-gitlab"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/pkg/tableprinter"
	"gitlab.com/gitlab-org/cli/pkg/utils"
)

type Options struct {
	OrderBy string
	Sort    string
	PerPage int
	Page    int

	BaseRepo   func() (glrepo.Interface, error)
	HTTPClient func() (*gitlab.Client, error)
	IO         *iostreams.IOStreams
}

func NewCmdContributors(f *cmdutils.Factory) *cobra.Command {
	opts := &Options{
		IO: f.IO,
	}
	repoContributorsCmd := &cobra.Command{
		Use:   "contributors",
		Short: `Get repository contributors list.`,
		Example: heredoc.Doc(`
	$ glab repo contributors

	# Supports overriding the repository.
	$ glab repo contributors -R gitlab-com/www-gitlab-com
	`),
		Args:    cobra.ExactArgs(0),
		Aliases: []string{"users"},
		RunE: func(cmd *cobra.Command, args []string) error {
			// Support `-R, --repo` override
			opts.BaseRepo = f.BaseRepo
			opts.HTTPClient = f.HttpClient

			return runE(opts)
		},
	}

	cmdutils.EnableRepoOverride(repoContributorsCmd, f)

	repoContributorsCmd.Flags().StringVarP(&opts.OrderBy, "order", "o", "commits", "Return contributors ordered by name, email, or commits (orders by commit date) fields.")
	repoContributorsCmd.Flags().StringVarP(&opts.Sort, "sort", "s", "", "Return contributors. Sort options: asc, desc.")
	repoContributorsCmd.Flags().IntVarP(&opts.Page, "page", "p", 1, "Page number.")
	repoContributorsCmd.Flags().IntVarP(&opts.PerPage, "per-page", "P", 30, "Number of items to list per page.")
	return repoContributorsCmd
}

func runE(opts *Options) error {
	var err error
	c := opts.IO.Color()

	apiClient, err := opts.HTTPClient()
	if err != nil {
		return err
	}

	repo, err := opts.BaseRepo()
	if err != nil {
		return err
	}

	if opts.OrderBy == "commits" && opts.Sort == "" {
		opts.Sort = "desc"
	}

	l := &gitlab.ListContributorsOptions{
		OrderBy: gitlab.Ptr(opts.OrderBy),
	}
	if opts.Sort != "" {
		l.Sort = gitlab.Ptr(opts.Sort)
	}
	l.PerPage = opts.PerPage
	l.Page = opts.Page

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

	fmt.Fprintf(opts.IO.StdOut, "%s\n%s\n", title.Describe(), table.String())
	return err
}
