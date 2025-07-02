package list

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"gitlab.com/gitlab-org/cli/internal/iostreams"

	"github.com/MakeNowJust/heredoc/v2"
	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/internal/glrepo"

	"gitlab.com/gitlab-org/cli/internal/api"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/commands/issuable"
	"gitlab.com/gitlab-org/cli/internal/commands/issue/issueutils"
	"gitlab.com/gitlab-org/cli/internal/utils"

	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

type ListOptions struct {
	// metadata
	Assignee    string
	NotAssignee string
	Author      string
	NotAuthor   string
	Labels      []string
	NotLabels   []string
	Milestone   string
	Mine        bool
	Search      string
	Group       string
	Epic        int
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

	IO        *iostreams.IOStreams
	BaseRepo  func() (glrepo.Interface, error)
	apiClient func(repoHost string, cfg config.Config) (*api.Client, error)
	config    config.Config

	JSONOutput bool
}

func NewCmdList(f cmdutils.Factory, runE func(opts *ListOptions) error, issueType issuable.IssueType) *cobra.Command {
	opts := &ListOptions{
		IO:        f.IO(),
		BaseRepo:  f.BaseRepo,
		apiClient: f.ApiClient,
		config:    f.Config(),
		IssueType: string(issueType),
	}

	issueListCmd := &cobra.Command{
		Use:     "list [flags]",
		Short:   fmt.Sprintf(`List project %ss.`, issueType),
		Long:    ``,
		Aliases: []string{"ls"},
		Example: heredoc.Doc(fmt.Sprintf(`
			$ glab %[1]s list --all
			$ glab %[1]s ls --all
			$ glab %[1]s list --assignee=@me
			$ glab %[1]s list --milestone release-2.0.0 --opened
		`, issueType)),
		Args: cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
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

			group, err := cmdutils.GroupOverride(cmd)
			if err != nil {
				return err
			}
			opts.Group = group

			if opts.Epic != 0 && opts.Group == "" {
				repo, err := opts.BaseRepo()
				if err != nil {
					return err
				}
				opts.Group = repo.RepoOwner()
			}
			if opts.Epic != 0 && opts.Group == "" {
				return cmdutils.FlagError{
					Err: errors.New("flag --epic requires flag --group"),
				}
			}

			// The underlying API, ListEpicIssues, does not support filtering, so we do the filtering client-side.
			// That means to implement pagination, we'd still need to request all previous issues and filter them.
			// That means client side pagination is more expensive (O(n^2)) than requesting all issues belonging to an epic (O(n)).
			if opts.Epic != 0 && opts.Page > 1 {
				return cmdutils.FlagError{
					Err: errors.New("--epic does not support the --page flag"),
				}
			}

			if runE != nil {
				return runE(opts)
			}

			return listRun(opts)
		},
	}
	cmdutils.EnableRepoOverride(issueListCmd, f)
	issueListCmd.Flags().StringVarP(&opts.Assignee, "assignee", "a", "", fmt.Sprintf("Filter %s by assignee <username>.", issueType))
	issueListCmd.Flags().StringVar(&opts.NotAssignee, "not-assignee", "", fmt.Sprintf("Filter %s by not being assigned to <username>.", issueType))
	issueListCmd.Flags().StringVar(&opts.Author, "author", "", fmt.Sprintf("Filter %s by author <username>.", issueType))
	issueListCmd.Flags().StringVar(&opts.NotAuthor, "not-author", "", fmt.Sprintf("Filter %s by not being by author(s) <username>.", issueType))
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
	issueListCmd.Flags().IntVarP(&opts.Epic, "epic", "e", 0, "List issues belonging to a given epic (requires --group, no pagination support).")
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
	// NOTE: this command can not only be used for projects,
	// so we have to manually check for the base repo, it it doesn't exist,
	// we bootstrap the client with the default hostname.
	var repoHost string
	if baseRepo, err := opts.BaseRepo(); err == nil {
		repoHost = baseRepo.RepoHost()
	}
	apiClient, err := opts.apiClient(repoHost, opts.config)
	if err != nil {
		return err
	}
	client := apiClient.Lab()

	listOpts := &gitlab.ListProjectIssuesOptions{
		State: gitlab.Ptr(opts.State),
		In:    gitlab.Ptr(opts.In),
	}
	listOpts.Page = 1
	listOpts.PerPage = 30

	if opts.Assignee == "" && opts.Mine {
		opts.Assignee = "@me"
	}

	if opts.Assignee != "" {
		uid, err := userID(client, opts.Assignee)
		if err != nil {
			return err
		}

		listOpts.AssigneeID = gitlab.Ptr(uid)
	}

	if opts.NotAssignee != "" {
		uid, err := userID(client, opts.NotAssignee)
		if err != nil {
			return err
		}
		listOpts.NotAssigneeID = gitlab.Ptr(uid)
	}

	if opts.Author != "" {
		uid, err := userID(client, opts.Author)
		if err != nil {
			return err
		}
		listOpts.AuthorID = gitlab.Ptr(uid)
	}

	if opts.NotAuthor != "" {
		uid, err := userID(client, opts.NotAuthor)
		if err != nil {
			return err
		}
		listOpts.NotAuthorID = gitlab.Ptr(uid)
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
	} else {
		listOpts.PerPage = api.DefaultListLimit
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
	switch {
	case opts.Epic != 0:
		issues, err = listEpicIssues(client, opts, listOpts)
		if err != nil {
			return err
		}
		title.RepoName = fmt.Sprintf("%s&%d", opts.Group, opts.Epic)

	case opts.Group != "":
		issues, _, err = client.Issues.ListGroupIssues(opts.Group, projectListIssueOptionsToGroup(listOpts))
		if err != nil {
			return err
		}
		title.RepoName = opts.Group

	default:
		repo, err := opts.BaseRepo()
		if err != nil {
			return err
		}

		issues, _, err = client.Issues.ListProjectIssues(repo.FullName(), listOpts)
		if err != nil {
			return err
		}
		title.RepoName = repo.FullName()
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

	fmt.Fprintf(opts.IO.StdOut, "%s\n%s\n", title.Describe(), issueutils.DisplayIssueList(opts.IO, issues, title.RepoName))
	return nil
}

func userID(client *gitlab.Client, username string) (int, error) {
	if username == "@me" {
		me, _, err := client.Users.CurrentUser()
		if err != nil {
			return 0, err
		}
		return me.ID, nil
	}

	u, err := api.UserByName(client, username)
	if err != nil {
		return 0, err
	}
	return u.ID, nil
}

// listEpicIssues is a wrapper around the API call of the same name.
// Since the GitLab API doesn't support filtering for this method, it implements client-side filtering of issues instead.
func listEpicIssues(client *gitlab.Client, opts *ListOptions, projListOpts *gitlab.ListProjectIssuesOptions) ([]*gitlab.Issue, error) {
	var (
		listOpts = gitlab.ListOptions{
			Page: 1,
		}
		issues []*gitlab.Issue
	)

	maxIssues := opts.PerPage
	if maxIssues <= 0 {
		maxIssues = api.DefaultListLimit
	}

	listOpts.PerPage = min(maxIssues, api.MaxPerPage)

	for {
		is, req, err := client.EpicIssues.ListEpicIssues(opts.Group, opts.Epic, &listOpts) //nolint:staticcheck
		if err != nil {
			return nil, err
		}

		// The "list issues for an epic" api doesn't support filtering, requiring client-side filtering.
		is = filterIssues(is, projListOpts)

		// If the number of issues exceeds the page size, trim the list.
		if len(issues)+len(is) > maxIssues {
			is = is[:maxIssues-len(issues)]
		}

		issues = append(issues, is...)

		if len(issues) >= maxIssues || req.NextPage == 0 {
			break
		}

		listOpts.Page = req.NextPage
	}

	return issues, nil
}

func filterIssues(issues []*gitlab.Issue, opts *gitlab.ListProjectIssuesOptions) []*gitlab.Issue {
	var ret []*gitlab.Issue

	for _, issue := range issues {
		if isMatch(issue, opts) {
			ret = append(ret, issue)
		}
	}

	return ret
}

func isMatch(issue *gitlab.Issue, opts *gitlab.ListProjectIssuesOptions) bool {
	if opts.AssigneeID != nil && !hasAssignee(issue, *opts.AssigneeID) {
		return false
	}
	if opts.NotAssigneeID != nil && hasAssignee(issue, *opts.NotAssigneeID) {
		return false
	}
	if opts.AuthorID != nil && (issue.Author == nil || issue.Author.ID != *opts.AuthorID) {
		return false
	}
	if opts.NotAuthorID != nil && issue.Author != nil && issue.Author.ID == *opts.NotAuthorID {
		return false
	}
	if opts.Labels != nil && !hasAllLabels(issue, []string(*opts.Labels)) {
		return false
	}
	if opts.NotLabels != nil && hasAnyLabel(issue, []string(*opts.NotLabels)) {
		return false
	}
	if opts.Milestone != nil && (issue.Milestone == nil || !strings.EqualFold(issue.Milestone.Title, *opts.Milestone)) {
		return false
	}
	if opts.Search != nil && !strings.Contains(strings.ToLower(issue.Title), strings.ToLower(*opts.Search)) {
		return false
	}
	if opts.IterationID != nil && (issue.Iteration == nil || issue.Iteration.ID != *opts.IterationID) {
		return false
	}

	if !stateMatches(issue, opts) {
		return false
	}
	if opts.Confidential != nil && *opts.Confidential != issue.Confidential {
		return false
	}

	return true
}

func hasAssignee(issue *gitlab.Issue, userID int) bool {
	for _, assignee := range issue.Assignees {
		if assignee.ID == userID {
			return true
		}
	}

	return false
}

func stateMatches(issue *gitlab.Issue, opts *gitlab.ListProjectIssuesOptions) bool {
	switch {
	case opts.State == nil:
		return true
	case *opts.State == "all":
		return true
	default:
		return *opts.State == issue.State
	}
}

func hasAllLabels(issue *gitlab.Issue, labels []string) bool {
	issueLabels := make(map[string]bool)
	for _, l := range issue.Labels {
		issueLabels[l] = true
	}

	for _, l := range labels {
		if _, ok := issueLabels[l]; !ok {
			return false
		}
	}

	return true
}

func hasAnyLabel(issue *gitlab.Issue, labels []string) bool {
	issueLabels := make(map[string]bool)
	for _, l := range issue.Labels {
		issueLabels[l] = true
	}

	for _, l := range labels {
		if _, ok := issueLabels[l]; ok {
			return true
		}
	}

	return false
}

func projectListIssueOptionsToGroup(l *gitlab.ListProjectIssuesOptions) *gitlab.ListGroupIssuesOptions {
	var assigneeID *gitlab.AssigneeIDValue
	if l.AssigneeID != nil {
		assigneeID = gitlab.AssigneeID(*l.AssigneeID)
	}
	return &gitlab.ListGroupIssuesOptions{
		ListOptions:        l.ListOptions,
		State:              l.State,
		Labels:             l.Labels,
		NotLabels:          l.NotLabels,
		WithLabelDetails:   l.WithLabelDetails,
		IIDs:               l.IIDs,
		Milestone:          l.Milestone,
		Scope:              l.Scope,
		AuthorID:           l.AuthorID,
		NotAuthorID:        l.NotAuthorID,
		AssigneeID:         assigneeID,
		NotAssigneeID:      l.NotAssigneeID,
		AssigneeUsername:   l.AssigneeUsername,
		MyReactionEmoji:    l.MyReactionEmoji,
		NotMyReactionEmoji: l.NotMyReactionEmoji,
		OrderBy:            l.OrderBy,
		Sort:               l.Sort,
		Search:             l.Search,
		In:                 l.In,
		CreatedAfter:       l.CreatedAfter,
		CreatedBefore:      l.CreatedBefore,
		UpdatedAfter:       l.UpdatedAfter,
		UpdatedBefore:      l.UpdatedBefore,
		IssueType:          l.IssueType,
	}
}
