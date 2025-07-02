package view

import (
	"encoding/json"
	"fmt"
	"strings"

	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/internal/iostreams"

	"gitlab.com/gitlab-org/cli/internal/api"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	issuableView "gitlab.com/gitlab-org/cli/internal/commands/issuable/view"
	"gitlab.com/gitlab-org/cli/internal/commands/mr/mrutils"
	"gitlab.com/gitlab-org/cli/internal/utils"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

var listMRNotes = func(client *gitlab.Client, projectID any, mrID int, opts *gitlab.ListMergeRequestNotesOptions) ([]*gitlab.Note, error) {
	if opts.PerPage == 0 {
		opts.PerPage = api.DefaultListLimit
	}

	notes, _, err := client.Notes.ListMergeRequestNotes(projectID, mrID, opts)
	if err != nil {
		return notes, err
	}

	return notes, nil
}

type options struct {
	showComments   bool
	showSystemLogs bool
	openInBrowser  bool
	outputFormat   string

	commentPageNujmber int
	commentLimit       int

	io         *iostreams.IOStreams
	httpClient func() (*gitlab.Client, error)
	config     func() config.Config
}

type MRWithNotes struct {
	*gitlab.MergeRequest
	Notes []*gitlab.Note
}

func NewCmdView(f cmdutils.Factory) *cobra.Command {
	opts := &options{
		io:         f.IO(),
		httpClient: f.HttpClient,
		config:     f.Config,
	}
	mrViewCmd := &cobra.Command{
		Use:     "view {<id> | <branch>}",
		Short:   `Display the title, body, and other information about a merge request.`,
		Long:    ``,
		Aliases: []string{"show"},
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return opts.run(f, args)
		},
	}

	mrViewCmd.Flags().BoolVarP(&opts.showComments, "comments", "c", false, "Show merge request comments and activities.")
	mrViewCmd.Flags().BoolVarP(&opts.showSystemLogs, "system-logs", "s", false, "Show system activities and logs.")
	mrViewCmd.Flags().StringVarP(&opts.outputFormat, "output", "F", "text", "Format output as: text, json.")
	mrViewCmd.Flags().BoolVarP(&opts.openInBrowser, "web", "w", false, "Open merge request in a browser. Uses default browser or browser specified in BROWSER variable.")
	mrViewCmd.Flags().IntVarP(&opts.commentPageNujmber, "page", "p", 0, "Page number.")
	mrViewCmd.Flags().IntVarP(&opts.commentLimit, "per-page", "P", 20, "Number of items to list per page.")

	return mrViewCmd
}

func (o *options) run(f cmdutils.Factory, args []string) error {
	apiClient, err := o.httpClient()
	if err != nil {
		return err
	}

	mr, baseRepo, err := mrutils.MRFromArgsWithOpts(f, args, &gitlab.GetMergeRequestsOptions{
		IncludeDivergedCommitsCount: gitlab.Ptr(true),
		RenderHTML:                  gitlab.Ptr(true),
		IncludeRebaseInProgress:     gitlab.Ptr(true),
	}, "any")
	if err != nil {
		return err
	}

	// Optional: check for approval state of the MR (if the project supports it). In the event of a failure
	// for this step, move forward assuming MR approvals are not supported. See below.
	//
	// NOTE: the API documentation says that project details have `approvals_before_merge` for GitLab Premium
	// https://docs.gitlab.com/api/projects/#get-a-single-project. Unfortunately, the API client used
	// does not provide the necessary ability to determine if this value was present or not in the response JSON
	// since Project.ApprovalsBeforeMerge is a non-pointer type. Because of this, this step will either succeed
	// and show approval state or it will fail silently
	mrApprovals, _, err := apiClient.MergeRequestApprovals.GetApprovalState(baseRepo.FullName(), mr.IID) //nolint:ineffassign,staticcheck

	cfg := o.config()

	if o.openInBrowser { // open in browser if --web flag is specified
		if o.io.IsOutputTTY() {
			fmt.Fprintf(o.io.StdErr, "Opening %s in your browser.\n", utils.DisplayURL(mr.WebURL))
		}

		browser, _ := cfg.Get(baseRepo.RepoHost(), "browser")
		return utils.OpenInBrowser(mr.WebURL, browser)
	}

	notes := []*gitlab.Note{}

	if o.showComments {
		l := &gitlab.ListMergeRequestNotesOptions{
			Sort: gitlab.Ptr("asc"),
			ListOptions: gitlab.ListOptions{
				Page:    o.commentPageNujmber,
				PerPage: o.commentLimit,
			},
		}

		notes, err = listMRNotes(apiClient, baseRepo.FullName(), mr.IID, l)
		if err != nil {
			return err
		}
	}

	glamourStyle, _ := cfg.Get(baseRepo.RepoHost(), "glamour_style")
	o.io.ResolveBackgroundColor(glamourStyle)
	if err := o.io.StartPager(); err != nil {
		return err
	}
	defer o.io.StopPager()

	switch {
	case o.outputFormat == "json":
		printJSONMR(o, mr, notes)
	case o.io.IsOutputTTY():
		printTTYMRPreview(o, mr, mrApprovals, notes)
	default:
		printRawMRPreview(o, mr, notes)
	}
	return nil
}

func labelsList(mr *gitlab.MergeRequest) string {
	return strings.Join(mr.Labels, ", ")
}

func assigneesList(mr *gitlab.MergeRequest) string {
	assignees := utils.Map(mr.Assignees, func(a *gitlab.BasicUser) string {
		return a.Username
	})

	return strings.Join(assignees, ", ")
}

func reviewersList(mr *gitlab.MergeRequest) string {
	reviewers := utils.Map(mr.Reviewers, func(r *gitlab.BasicUser) string {
		return r.Username
	})

	return strings.Join(reviewers, ", ")
}

func mrState(c *iostreams.ColorPalette, mr *gitlab.MergeRequest) string {
	switch mr.State {
	case "opened":
		return c.Green("open")
	case "merged":
		return c.Blue(mr.State)
	default:
		return c.Red(mr.State)
	}
}

func printTTYMRPreview(opts *options, mr *gitlab.MergeRequest, mrApprovals *gitlab.MergeRequestApprovalState, notes []*gitlab.Note) {
	c := opts.io.Color()
	out := opts.io.StdOut
	mrTimeAgo := utils.TimeToPrettyTimeAgo(*mr.CreatedAt)
	// Header
	fmt.Fprint(out, mrState(c, mr))
	fmt.Fprintf(out, c.Gray(" • opened by %s %s\n"), mr.Author.Username, mrTimeAgo)
	fmt.Fprint(out, mr.Title)
	fmt.Fprintf(out, c.Gray(" !%d"), mr.IID)
	fmt.Fprintln(out)

	// Description
	if mr.Description != "" {
		mr.Description, _ = utils.RenderMarkdown(mr.Description, opts.io.BackgroundColor())
		fmt.Fprintln(out, mr.Description)
	}

	fmt.Fprintf(out, c.Gray("\n%d upvotes • %d downvotes • %d comments\n"), mr.Upvotes, mr.Downvotes, mr.UserNotesCount)

	// Meta information
	if labels := labelsList(mr); labels != "" {
		fmt.Fprint(out, c.Bold("Labels: "))
		fmt.Fprintln(out, labels)
	}
	if assignees := assigneesList(mr); assignees != "" {
		fmt.Fprint(out, c.Bold("Assignees: "))
		fmt.Fprintln(out, assignees)
	}
	if reviewers := reviewersList(mr); reviewers != "" {
		fmt.Fprint(out, c.Bold("Reviewers: "))
		fmt.Fprintln(out, reviewers)
	}
	if mr.Milestone != nil {
		fmt.Fprint(out, c.Bold("Milestone: "))
		fmt.Fprintln(out, mr.Milestone.Title)
	}
	if mr.State == "closed" {
		fmt.Fprintf(out, "Closed by: %s %s\n", mr.ClosedBy.Username, mrTimeAgo)
	}
	if mr.Pipeline != nil {
		fmt.Fprint(out, c.Bold("Pipeline status: "))
		var status string
		switch s := mr.Pipeline.Status; s {
		case "failed":
			status = c.Red(s)
		case "success":
			status = c.Green(s)
		default:
			status = c.Gray(s)
		}
		fmt.Fprintf(out, "%s (View pipeline with `%s`)\n", status, c.Bold("glab ci view "+mr.SourceBranch))

		if mr.MergeWhenPipelineSucceeds && mr.Pipeline.Status != "success" {
			fmt.Fprintf(out, "%s Requires pipeline to succeed before merging.\n", c.WarnIcon())
		}
	}
	if mrApprovals != nil {
		fmt.Fprintln(out, c.Bold("Approvals status:"))
		mrutils.PrintMRApprovalState(opts.io, mrApprovals)
	}
	fmt.Fprintf(out, "%s This merge request has %s changes.\n", c.GreenCheck(), c.Yellow(mr.ChangesCount))
	if mr.State == "merged" && mr.MergedBy != nil { //nolint:staticcheck
		fmt.Fprintf(out, "%s The changes were merged into %s by %s %s.\n", c.GreenCheck(), mr.TargetBranch, mr.MergedBy.Name, utils.TimeToPrettyTimeAgo(*mr.MergedAt)) //nolint:staticcheck
	}

	if mr.HasConflicts {
		fmt.Fprintf(out, c.Red("%s This branch has conflicts that must be resolved.\n"), c.FailedIcon())
	}

	// Comments
	if opts.showComments {
		fmt.Fprintln(out, heredoc.Doc(`
			--------------------------------------------
			Comments / Notes
			--------------------------------------------
			`))
		if len(notes) > 0 {
			for _, note := range notes {
				if note.System && !opts.showSystemLogs {
					continue
				}
				createdAt := utils.TimeToPrettyTimeAgo(*note.CreatedAt)
				fmt.Fprint(out, note.Author.Username)
				if note.System {
					fmt.Fprintf(out, " %s ", note.Body)
					fmt.Fprintln(out, c.Gray(createdAt))
				} else {
					body, _ := utils.RenderMarkdown(note.Body, opts.io.BackgroundColor())
					fmt.Fprint(out, " commented ")
					fmt.Fprintf(out, c.Gray("%s\n"), createdAt)
					fmt.Fprintln(out, utils.Indent(body, " "))
				}
				fmt.Fprintln(out)
			}
		} else {
			fmt.Fprintln(out, "This merge request has no comments.")
		}
	}

	fmt.Fprintln(out)
	fmt.Fprintf(out, c.Gray("View this merge request on GitLab: %s\n"), mr.WebURL)
}

func printRawMRPreview(opts *options, mr *gitlab.MergeRequest, notes []*gitlab.Note) {
	fmt.Fprint(opts.io.StdOut, rawMRPreview(opts, mr, notes))
}

func rawMRPreview(opts *options, mr *gitlab.MergeRequest, notes []*gitlab.Note) string {
	var out string

	assignees := assigneesList(mr)
	reviewers := reviewersList(mr)
	labels := labelsList(mr)

	out += fmt.Sprintf("title:\t%s\n", mr.Title)
	out += fmt.Sprintf("state:\t%s\n", mrState(opts.io.Color(), mr))
	out += fmt.Sprintf("author:\t%s\n", mr.Author.Username)
	out += fmt.Sprintf("labels:\t%s\n", labels)
	out += fmt.Sprintf("assignees:\t%s\n", assignees)
	out += fmt.Sprintf("reviewers:\t%s\n", reviewers)
	out += fmt.Sprintf("comments:\t%d\n", mr.UserNotesCount)
	if mr.Milestone != nil {
		out += fmt.Sprintf("milestone:\t%s\n", mr.Milestone.Title)
	}
	out += fmt.Sprintf("number:\t%d\n", mr.IID)
	out += fmt.Sprintf("url:\t%s\n", mr.WebURL)
	out += "--\n"
	out += fmt.Sprintf("%s\n", mr.Description)

	out += issuableView.RawIssuableNotes(notes, opts.showComments, opts.showSystemLogs, "merge request")

	return out
}

func printJSONMR(opts *options, mr *gitlab.MergeRequest, notes []*gitlab.Note) {
	if opts.showComments {
		extendedMR := MRWithNotes{mr, notes}
		mrJSON, _ := json.Marshal(extendedMR)
		fmt.Fprintln(opts.io.StdOut, string(mrJSON))
	} else {
		mrJSON, _ := json.Marshal(mr)
		fmt.Fprintln(opts.io.StdOut, string(mrJSON))
	}
}
