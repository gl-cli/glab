package view

import (
	"encoding/json"
	"fmt"
	"strings"

	"gitlab.com/gitlab-org/cli/pkg/iostreams"

	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	issuableView "gitlab.com/gitlab-org/cli/commands/issuable/view"
	"gitlab.com/gitlab-org/cli/commands/mr/mrutils"
	"gitlab.com/gitlab-org/cli/pkg/utils"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	"github.com/xanzy/go-gitlab"
)

type ViewOpts struct {
	ShowComments   bool
	ShowSystemLogs bool
	OpenInBrowser  bool
	OutputFormat   string

	CommentPageNumber int
	CommentLimit      int

	IO *iostreams.IOStreams
}

type MRWithNotes struct {
	*gitlab.MergeRequest
	Notes []*gitlab.Note
}

func NewCmdView(f *cmdutils.Factory) *cobra.Command {
	opts := &ViewOpts{
		IO: f.IO,
	}
	mrViewCmd := &cobra.Command{
		Use:     "view {<id> | <branch>}",
		Short:   `Display the title, body, and other information about a merge request.`,
		Long:    ``,
		Aliases: []string{"show"},
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			apiClient, err := f.HttpClient()
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
			// https://docs.gitlab.com/ee/api/projects.html#get-single-project. Unfortunately, the API client used
			// does not provide the necessary ability to determine if this value was present or not in the response JSON
			// since Project.ApprovalsBeforeMerge is a non-pointer type. Because of this, this step will either succeed
			// and show approval state or it will fail silently
			mrApprovals, _ := api.GetMRApprovalState(apiClient, baseRepo.FullName(), mr.IID)

			cfg, _ := f.Config()

			if opts.OpenInBrowser { // open in browser if --web flag is specified
				if f.IO.IsOutputTTY() {
					fmt.Fprintf(f.IO.StdErr, "Opening %s in your browser.\n", utils.DisplayURL(mr.WebURL))
				}

				browser, _ := cfg.Get(baseRepo.RepoHost(), "browser")
				return utils.OpenInBrowser(mr.WebURL, browser)
			}

			notes := []*gitlab.Note{}

			if opts.ShowComments {
				l := &gitlab.ListMergeRequestNotesOptions{
					Sort: gitlab.Ptr("asc"),
				}
				l.Page = opts.CommentPageNumber
				l.PerPage = opts.CommentLimit

				notes, err = api.ListMRNotes(apiClient, baseRepo.FullName(), mr.IID, l)
				if err != nil {
					return err
				}
			}

			glamourStyle, _ := cfg.Get(baseRepo.RepoHost(), "glamour_style")
			f.IO.ResolveBackgroundColor(glamourStyle)
			if err := f.IO.StartPager(); err != nil {
				return err
			}
			defer f.IO.StopPager()

			if opts.OutputFormat == "json" {
				return printJSONMR(opts, mr, notes)
			}
			if f.IO.IsOutputTTY() {
				return printTTYMRPreview(opts, mr, mrApprovals, notes)
			}
			return printRawMRPreview(opts, mr, notes)
		},
	}

	mrViewCmd.Flags().BoolVarP(&opts.ShowComments, "comments", "c", false, "Show merge request comments and activities.")
	mrViewCmd.Flags().BoolVarP(&opts.ShowSystemLogs, "system-logs", "s", false, "Show system activities and logs.")
	mrViewCmd.Flags().StringVarP(&opts.OutputFormat, "output", "F", "text", "Format output as: text, json.")
	mrViewCmd.Flags().BoolVarP(&opts.OpenInBrowser, "web", "w", false, "Open merge request in a browser. Uses default browser or browser specified in BROWSER variable.")
	mrViewCmd.Flags().IntVarP(&opts.CommentPageNumber, "page", "p", 0, "Page number.")
	mrViewCmd.Flags().IntVarP(&opts.CommentLimit, "per-page", "P", 20, "Number of items to list per page.")

	return mrViewCmd
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

func mrState(c *iostreams.ColorPalette, mr *gitlab.MergeRequest) (mrState string) {
	if mr.State == "opened" {
		mrState = c.Green("open")
	} else if mr.State == "merged" {
		mrState = c.Blue(mr.State)
	} else {
		mrState = c.Red(mr.State)
	}

	return mrState
}

func printTTYMRPreview(opts *ViewOpts, mr *gitlab.MergeRequest, mrApprovals *gitlab.MergeRequestApprovalState, notes []*gitlab.Note) error {
	c := opts.IO.Color()
	out := opts.IO.StdOut
	mrTimeAgo := utils.TimeToPrettyTimeAgo(*mr.CreatedAt)
	// Header
	fmt.Fprint(out, mrState(c, mr))
	fmt.Fprintf(out, c.Gray(" • opened by %s %s\n"), mr.Author.Username, mrTimeAgo)
	fmt.Fprint(out, mr.Title)
	fmt.Fprintf(out, c.Gray(" !%d"), mr.IID)
	fmt.Fprintln(out)

	// Description
	if mr.Description != "" {
		mr.Description, _ = utils.RenderMarkdown(mr.Description, opts.IO.BackgroundColor())
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
		mrutils.PrintMRApprovalState(opts.IO, mrApprovals)
	}
	fmt.Fprintf(out, "%s This merge request has %s changes.\n", c.GreenCheck(), c.Yellow(mr.ChangesCount))
	if mr.State == "merged" && mr.MergedBy != nil {
		fmt.Fprintf(out, "%s The changes were merged into %s by %s %s.\n", c.GreenCheck(), mr.TargetBranch, mr.MergedBy.Name, utils.TimeToPrettyTimeAgo(*mr.MergedAt))
	}

	if mr.HasConflicts {
		fmt.Fprintf(out, c.Red("%s This branch has conflicts that must be resolved.\n"), c.FailedIcon())
	}

	// Comments
	if opts.ShowComments {
		fmt.Fprintln(out, heredoc.Doc(`
			--------------------------------------------
			Comments / Notes
			--------------------------------------------
			`))
		if len(notes) > 0 {
			for _, note := range notes {
				if note.System && !opts.ShowSystemLogs {
					continue
				}
				createdAt := utils.TimeToPrettyTimeAgo(*note.CreatedAt)
				fmt.Fprint(out, note.Author.Username)
				if note.System {
					fmt.Fprintf(out, " %s ", note.Body)
					fmt.Fprintln(out, c.Gray(createdAt))
				} else {
					body, _ := utils.RenderMarkdown(note.Body, opts.IO.BackgroundColor())
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

	return nil
}

func printRawMRPreview(opts *ViewOpts, mr *gitlab.MergeRequest, notes []*gitlab.Note) error {
	fmt.Fprint(opts.IO.StdOut, rawMRPreview(opts, mr, notes))

	return nil
}

func rawMRPreview(opts *ViewOpts, mr *gitlab.MergeRequest, notes []*gitlab.Note) string {
	var out string

	assignees := assigneesList(mr)
	reviewers := reviewersList(mr)
	labels := labelsList(mr)

	out += fmt.Sprintf("title:\t%s\n", mr.Title)
	out += fmt.Sprintf("state:\t%s\n", mrState(opts.IO.Color(), mr))
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

	out += issuableView.RawIssuableNotes(notes, opts.ShowComments, opts.ShowSystemLogs, "merge request")

	return out
}

func printJSONMR(opts *ViewOpts, mr *gitlab.MergeRequest, notes []*gitlab.Note) error {
	if opts.ShowComments {
		extendedMR := MRWithNotes{mr, notes}
		mrJSON, _ := json.Marshal(extendedMR)
		fmt.Fprintln(opts.IO.StdOut, string(mrJSON))
	} else {
		mrJSON, _ := json.Marshal(mr)
		fmt.Fprintln(opts.IO.StdOut, string(mrJSON))
	}
	return nil
}
