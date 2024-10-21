package list

import (
	"encoding/json"
	"errors"
	"fmt"

	"gitlab.com/gitlab-org/cli/pkg/iostreams"

	"github.com/MakeNowJust/heredoc/v2"
	"gitlab.com/gitlab-org/cli/internal/glrepo"

	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/commands/flag"
	"gitlab.com/gitlab-org/cli/commands/issuable"
	"gitlab.com/gitlab-org/cli/commands/issue/issueutils"
	"gitlab.com/gitlab-org/cli/pkg/utils"

	"github.com/spf13/cobra"
	"github.com/xanzy/go-gitlab"
)

type ListOptions struct {
	// metadata
	Assignee    string
	NotAssignee []string
	Author      string
	NotAuthor   []string
	Labels      []string
	NotLabels   []string
	Milestone   string
	Mine        bool
	Search      string
	Group       string
	IssueType   string
	Iteration   int

	// issue states
	State        string
	Closed       bool
	Opened       bool
	All          bool
	Confidential bool

	// Pagination
	Page    int
	PerPage int

	// Other
	In string

	// display opts
	ListType       string
	TitleQualifier string
	OutputFormat   string
	Output         string

	IO         *iostreams.IOStreams
	BaseRepo   func() (glrepo.Interface, error)
	HTTPClient func() (*gitlab.Client, error)

	JSONOutput bool
}

func NewCmdList(f *cmdutils.Factory, runE func(opts *ListOptions) error, issueType issuable.IssueType) *cobra.Command {
	opts := &ListOptions{
		IO:        f.IO,
		IssueType: string(issueType),
	}

	issueListCmd := &cobra.Command{
		Use:     "list [flags]",
		Short:   fmt.Sprintf(`List project %ss.`, issueType),
		Long:    ``,
		Aliases: []string{"ls"},
		Example: heredoc.Doc(fmt.Sprintf(`
			glab %[1]s list --all
			glab %[1]s ls --all
			glab %[1]s list --assignee=@me
			glab %[1]s list --milestone release-2.0.0 --opened
		`, issueType)),
		Args: cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			// support repo override
			opts.BaseRepo = f.BaseRepo
			opts.HTTPClient = f.HttpClient

			if len(opts.Labels) != 0 && len(opts.NotLabels) != 0 {
				return cmdutils.FlagError{
					Err: errors.New("flags --label and --not-label are mutually exclusive."),
				}
			}

			if opts.Author != "" && len(opts.NotAuthor) != 0 {
				return cmdutils.FlagError{
					Err: errors.New("flags --author and --not-author are mutually exclusive."),
				}
			}

			if opts.Assignee != "" && len(opts.NotAssignee) != 0 {
				return cmdutils.FlagError{
					Err: errors.New("flags --assignee and --not-assignee are mutually exclusive."),
				}
			}

			if opts.All {
				opts.State = "all"
			} else if opts.Closed {
				opts.State = "closed"
				opts.TitleQualifier = "closed"
			} else {
				opts.State = "opened"
				opts.TitleQualifier = "open"
			}

			group, err := flag.GroupOverride(cmd)
			if err != nil {
				return err
			}
			opts.Group = group

			if runE != nil {
				return runE(opts)
			}

			return listRun(opts)
		},
	}
	cmdutils.EnableRepoOverride(issueListCmd, f)
	issueListCmd.Flags().StringVarP(&opts.Assignee, "assignee", "a", "", fmt.Sprintf("Filter %s by assignee <username>.", issueType))
	issueListCmd.Flags().StringSliceVar(&opts.NotAssignee, "not-assignee", []string{}, fmt.Sprintf("Filter %s by not being assigneed to <username>.", issueType))
	issueListCmd.Flags().StringVar(&opts.Author, "author", "", fmt.Sprintf("Filter %s by author <username>.", issueType))
	issueListCmd.Flags().StringSliceVar(&opts.NotAuthor, "not-author", []string{}, "Filter by not being by author(s) <username>.")
	issueListCmd.Flags().StringVar(&opts.Search, "search", "", "Search <string> in the fields defined by '--in'.")
	issueListCmd.Flags().StringVar(&opts.In, "in", "title,description", "search in: title, description.")
	issueListCmd.Flags().StringSliceVarP(&opts.Labels, "label", "l", []string{}, fmt.Sprintf("Filter %s by label <name>.", issueType))
	issueListCmd.Flags().StringSliceVar(&opts.NotLabels, "not-label", []string{}, fmt.Sprintf("Filter %s by lack of label <name>.", issueType))
	issueListCmd.Flags().StringVarP(&opts.Milestone, "milestone", "m", "", fmt.Sprintf("Filter %s by milestone <id>.", issueType))
	issueListCmd.Flags().BoolVarP(&opts.All, "all", "A", false, fmt.Sprintf("Get all %ss.", issueType))
	issueListCmd.Flags().BoolVarP(&opts.Closed, "closed", "c", false, fmt.Sprintf("Get only closed %ss.", issueType))
	issueListCmd.Flags().BoolVarP(&opts.Confidential, "confidential", "C", false, fmt.Sprintf("Filter by confidential %ss.", issueType))
	issueListCmd.Flags().StringVarP(&opts.OutputFormat, "output-format", "F", "details", "Options: 'details', 'ids', 'urls'.")
	issueListCmd.Flags().StringVarP(&opts.Output, "output", "O", "text", "Options: 'text' or 'json'.")
	issueListCmd.Flags().IntVarP(&opts.Page, "page", "p", 1, "Page number.")
	issueListCmd.Flags().IntVarP(&opts.PerPage, "per-page", "P", 30, "Number of items to list per page.")
	issueListCmd.PersistentFlags().StringP("group", "g", "", "Select a group or subgroup. Ignored if a repo argument is set.")
	issueListCmd.MarkFlagsMutuallyExclusive("output", "output-format")

	if issueType == issuable.TypeIssue {
		issueListCmd.Flags().StringVarP(&opts.IssueType, "issue-type", "t", "", "Filter issue by its type. Options: issue, incident, test_case.")
		issueListCmd.Flags().IntVarP(&opts.Iteration, "iteration", "i", 0, "Filter issue by iteration <id>.")
	}

	issueListCmd.Flags().BoolP("opened", "o", false, fmt.Sprintf("Get only open %ss.", issueType))
	_ = issueListCmd.Flags().MarkHidden("opened")
	_ = issueListCmd.Flags().MarkDeprecated("opened", "default if --closed is not used.")

	issueListCmd.Flags().BoolVarP(&opts.Mine, "mine", "M", false, fmt.Sprintf("Filter only %ss assigned to me.", issueType))
	_ = issueListCmd.Flags().MarkHidden("mine")
	_ = issueListCmd.Flags().MarkDeprecated("mine", "use --assignee=@me")

	return issueListCmd
}

func listRun(opts *ListOptions) error {
	apiClient, err := opts.HTTPClient()
	if err != nil {
		return err
	}

	repo, err := opts.BaseRepo()
	if err != nil {
		return err
	}

	listOpts := &gitlab.ListProjectIssuesOptions{
		State: gitlab.Ptr(opts.State),
		In:    gitlab.Ptr(opts.In),
	}
	listOpts.Page = 1
	listOpts.PerPage = 30

	if opts.Assignee != "" || opts.Mine {
		if opts.Assignee == "@me" || opts.Mine {
			u, err := api.CurrentUser(nil)
			if err != nil {
				return err
			}
			opts.Assignee = u.Username
		}
		listOpts.AssigneeUsername = gitlab.Ptr(opts.Assignee)
	}
	if len(opts.NotAssignee) != 0 {
		u, err := api.UsersByNames(apiClient, opts.NotAssignee)
		if err != nil {
			return err
		}
		listOpts.NotAssigneeID = cmdutils.IDsFromUsers(u)
	}
	if opts.Author != "" {
		u, err := api.UserByName(apiClient, opts.Author)
		if err != nil {
			return err
		}
		listOpts.AuthorID = gitlab.Ptr(u.ID)
	}
	if len(opts.NotAuthor) != 0 {
		u, err := api.UsersByNames(apiClient, opts.NotAuthor)
		if err != nil {
			return err
		}
		listOpts.NotAuthorID = cmdutils.IDsFromUsers(u)
	}
	if opts.Search != "" {
		listOpts.Search = gitlab.Ptr(opts.Search)
		opts.ListType = "search"
	}
	if len(opts.Labels) != 0 {
		listOpts.Labels = (*gitlab.LabelOptions)(&opts.Labels)
		opts.ListType = "search"
	}
	if len(opts.NotLabels) != 0 {
		listOpts.NotLabels = (*gitlab.LabelOptions)(&opts.NotLabels)
		opts.ListType = "search"
	}
	if opts.Milestone != "" {
		listOpts.Milestone = gitlab.Ptr(opts.Milestone)
		opts.ListType = "search"
	}
	if opts.Confidential {
		listOpts.Confidential = gitlab.Ptr(opts.Confidential)
		opts.ListType = "search"
	}
	if opts.Page != 0 {
		listOpts.Page = opts.Page
		opts.ListType = "search"
	}
	if opts.PerPage != 0 {
		listOpts.PerPage = opts.PerPage
		opts.ListType = "search"
	}

	issueType := "issue"
	if opts.IssueType != "" {
		listOpts.IssueType = gitlab.Ptr(opts.IssueType)
		opts.ListType = "search"
		issueType = opts.IssueType
	}
	if issueType == "issue" && opts.Iteration != 0 {
		listOpts.IterationID = gitlab.Ptr(opts.Iteration)
	}

	var issues []*gitlab.Issue
	title := utils.NewListTitle(fmt.Sprintf("%s %s", opts.TitleQualifier, issueType))
	title.RepoName = repo.FullName()
	if opts.Group != "" {
		issues, err = api.ListGroupIssues(apiClient, opts.Group, api.ProjectListIssueOptionsToGroup(listOpts))
		if err != nil {
			return err
		}
		title.RepoName = opts.Group
	} else {
		issues, err = api.ListIssues(apiClient, repo.FullName(), listOpts)
		if err != nil {
			return err
		}
	}

	title.Page = listOpts.Page
	title.ListActionType = opts.ListType
	title.CurrentPageTotal = len(issues)

	if opts.Output == "json" {
		issueListJSON, _ := json.Marshal(issues)
		fmt.Fprintln(opts.IO.StdOut, string(issueListJSON))
		return nil
	}

	if opts.OutputFormat == "ids" {
		for _, i := range issues {
			fmt.Fprintf(opts.IO.StdOut, "%d\n", i.IID)
		}
		return nil
	}

	if opts.OutputFormat == "urls" {
		for _, i := range issues {
			fmt.Fprintf(opts.IO.StdOut, "%s\n", i.WebURL)
		}
		return nil
	}

	if opts.IO.StartPager() != nil {
		return fmt.Errorf("failed to start pager: %q", err)
	}
	defer opts.IO.StopPager()

	fmt.Fprintf(opts.IO.StdOut, "%s\n%s\n", title.Describe(), issueutils.DisplayIssueList(opts.IO, issues, repo.FullName()))
	return nil
}
