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
	"gitlab.com/gitlab-org/cli/commands/mr/mrutils"
	"gitlab.com/gitlab-org/cli/pkg/utils"

	"github.com/spf13/cobra"
	"github.com/xanzy/go-gitlab"
)

type ListOptions struct {
	// metadata
	Assignee     []string
	Reviewer     []string
	Author       string
	Labels       []string
	NotLabels    []string
	Milestone    string
	SourceBranch string
	TargetBranch string
	Search       string
	Mine         bool
	Group        string

	// issue states
	State  string
	Closed bool
	Merged bool
	All    bool
	Draft  bool

	// Pagination
	Page         int
	PerPage      int
	OutputFormat string

	// display opts
	ListType       string
	TitleQualifier string

	IO         *iostreams.IOStreams
	BaseRepo   func() (glrepo.Interface, error)
	HTTPClient func() (*gitlab.Client, error)
}

func NewCmdList(f *cmdutils.Factory, runE func(opts *ListOptions) error) *cobra.Command {
	opts := &ListOptions{
		IO: f.IO,
	}

	mrListCmd := &cobra.Command{
		Use:     "list [flags]",
		Short:   `List merge requests.`,
		Long:    ``,
		Aliases: []string{"ls"},
		Example: heredoc.Doc(`
			glab mr list --all
			glab mr ls -a
			glab mr list --assignee=@me
			glab mr list --reviewer=@me
			glab mr list --source-branch=new-feature
			glab mr list --target-branch=main
			glab mr list --search "this adds feature X"
			glab mr list --label needs-review
			glab mr list --not-label waiting-maintainer-feedback,subsystem-x
			glab mr list -M --per-page 10
		`),
		Args: cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			// supports repo override
			opts.BaseRepo = f.BaseRepo
			opts.HTTPClient = f.HttpClient

			if len(opts.Labels) != 0 && len(opts.NotLabels) != 0 {
				return cmdutils.FlagError{
					Err: errors.New("flags --label and --not-label are mutually exclusive."),
				}
			}

			// check if any of the two or all of states flag are specified
			if opts.Closed && opts.Merged {
				return cmdutils.FlagError{
					Err: errors.New("specify either --closed or --merged. Use --all issues in all states."),
				}
			}
			if opts.All {
				opts.State = "all"
			} else if opts.Closed {
				opts.State = "closed"
				opts.TitleQualifier = opts.State
			} else if opts.Merged {
				opts.State = "merged"
				opts.TitleQualifier = opts.State
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

	cmdutils.EnableRepoOverride(mrListCmd, f)
	mrListCmd.Flags().StringSliceVarP(&opts.Labels, "label", "l", []string{}, "Filter merge request by label <name>.")
	mrListCmd.Flags().StringSliceVar(&opts.NotLabels, "not-label", []string{}, "Filter merge requests by not having label <name>.")
	mrListCmd.Flags().StringVar(&opts.Author, "author", "", "Filter merge request by author <username>.")
	mrListCmd.Flags().StringVarP(&opts.Milestone, "milestone", "m", "", "Filter merge request by milestone <id>.")
	mrListCmd.Flags().StringVarP(&opts.SourceBranch, "source-branch", "s", "", "Filter by source branch <name>.")
	mrListCmd.Flags().StringVarP(&opts.TargetBranch, "target-branch", "t", "", "Filter by target branch <name>.")
	mrListCmd.Flags().StringVar(&opts.Search, "search", "", "Filter by <string> in title and description.")
	mrListCmd.Flags().BoolVarP(&opts.All, "all", "A", false, "Get all merge requests.")
	mrListCmd.Flags().BoolVarP(&opts.Closed, "closed", "c", false, "Get only closed merge requests.")
	mrListCmd.Flags().BoolVarP(&opts.Merged, "merged", "M", false, "Get only merged merge requests.")
	mrListCmd.Flags().BoolVarP(&opts.Draft, "draft", "d", false, "Filter by draft merge requests.")
	mrListCmd.Flags().StringVarP(&opts.OutputFormat, "output", "F", "text", "Format output as: text, json.")
	mrListCmd.Flags().IntVarP(&opts.Page, "page", "p", 1, "Page number.")
	mrListCmd.Flags().IntVarP(&opts.PerPage, "per-page", "P", 30, "Number of items to list per page.")
	mrListCmd.Flags().StringSliceVarP(&opts.Assignee, "assignee", "a", []string{}, "Get only merge requests assigned to users.")
	mrListCmd.Flags().StringSliceVarP(&opts.Reviewer, "reviewer", "r", []string{}, "Get only merge requests with users as reviewer.")

	mrListCmd.Flags().BoolP("opened", "O", false, "Get only open merge requests.")
	_ = mrListCmd.Flags().MarkHidden("opened")
	_ = mrListCmd.Flags().MarkDeprecated("opened", "default value if neither --closed, --locked or --merged is used.")

	mrListCmd.Flags().BoolVarP(&opts.Mine, "mine", "", false, "Get only merge requests assigned to me.")
	_ = mrListCmd.Flags().MarkHidden("mine")
	_ = mrListCmd.Flags().MarkDeprecated("mine", "use --assignee=@me.")
	mrListCmd.PersistentFlags().StringP("group", "g", "", "Select a group/subgroup. This option is ignored if a repo argument is set.")

	return mrListCmd
}

func listRun(opts *ListOptions) error {
	var mergeRequests []*gitlab.MergeRequest

	apiClient, err := opts.HTTPClient()
	if err != nil {
		return err
	}

	repo, err := opts.BaseRepo()
	if err != nil {
		return err
	}

	l := &gitlab.ListProjectMergeRequestsOptions{
		State: gitlab.Ptr(opts.State),
	}
	jsonOutput := opts.OutputFormat == "json"
	if jsonOutput {
		l.Page = 0
		l.PerPage = 0
	} else {
		l.Page = 1
		l.PerPage = 30
	}

	if opts.Author != "" {
		u, err := api.UserByName(apiClient, opts.Author)
		if err != nil {
			return err
		}
		l.AuthorID = gitlab.Ptr(u.ID)
		opts.ListType = "search"
	}
	if opts.SourceBranch != "" {
		l.SourceBranch = gitlab.Ptr(opts.SourceBranch)
		opts.ListType = "search"
	}
	if opts.TargetBranch != "" {
		l.TargetBranch = gitlab.Ptr(opts.TargetBranch)
		opts.ListType = "search"
	}
	if opts.Search != "" {
		l.Search = gitlab.Ptr(opts.Search)
		opts.ListType = "search"
	}
	if len(opts.Labels) > 0 {
		l.Labels = (*gitlab.LabelOptions)(&opts.Labels)
		opts.ListType = "search"
	}
	if len(opts.NotLabels) > 0 {
		l.NotLabels = (*gitlab.LabelOptions)(&opts.NotLabels)
		opts.ListType = "search"
	}
	if opts.Milestone != "" {
		l.Milestone = gitlab.Ptr(opts.Milestone)
		opts.ListType = "search"
	}
	if opts.Page != 0 {
		l.Page = opts.Page
	}
	if opts.PerPage != 0 {
		l.PerPage = opts.PerPage
	}
	if opts.Draft {
		l.WIP = gitlab.Ptr("yes")
		opts.ListType = "search"
	}

	if opts.Mine {
		l.Scope = gitlab.Ptr("assigned_to_me")
		opts.ListType = "search"
	}

	assigneeIds := make([]int, 0)
	if len(opts.Assignee) > 0 {
		users, err := api.UsersByNames(apiClient, opts.Assignee)
		if err != nil {
			return err
		}
		for _, user := range users {
			assigneeIds = append(assigneeIds, user.ID)
		}
	}

	reviewerIds := make([]int, 0)
	if len(opts.Reviewer) > 0 {
		users, err := api.UsersByNames(apiClient, opts.Reviewer)
		if err != nil {
			return err
		}
		for _, user := range users {
			reviewerIds = append(reviewerIds, user.ID)
		}
	}
	title := utils.NewListTitle(opts.TitleQualifier + " merge request")
	title.RepoName = repo.FullName()

	if opts.Group != "" {
		mergeRequests, err = api.ListGroupMRs(apiClient, opts.Group, api.ProjectListMROptionsToGroup(l), api.WithMRAssignees(assigneeIds), api.WithMRReviewers(reviewerIds))
		title.RepoName = opts.Group
	} else {
		mergeRequests, err = api.ListMRs(apiClient, repo.FullName(), l, api.WithMRAssignees(assigneeIds), api.WithMRReviewers(reviewerIds))
	}
	if err != nil {
		return err
	}

	title.Page = l.Page
	title.ListActionType = opts.ListType
	title.CurrentPageTotal = len(mergeRequests)

	if jsonOutput {
		mrListJSON, _ := json.Marshal(mergeRequests)
		fmt.Fprintln(opts.IO.StdOut, string(mrListJSON))
	} else {
		if err = opts.IO.StartPager(); err != nil {
			return err
		}
		defer opts.IO.StopPager()
		fmt.Fprintf(opts.IO.StdOut, "%s\n%s\n", title.Describe(), mrutils.DisplayAllMRs(opts.IO, mergeRequests))
	}
	return nil
}
