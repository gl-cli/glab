package view

import (
	"encoding/json"
	"fmt"
	"strings"

	"gitlab.com/gitlab-org/cli/pkg/iostreams"

	"gitlab.com/gitlab-org/cli/commands/issuable"
	"gitlab.com/gitlab-org/cli/commands/issue/issueutils"

	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/pkg/utils"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	"github.com/xanzy/go-gitlab"
)

type IssueWithNotes struct {
	*gitlab.Issue
	Notes []*gitlab.Note
}

type ViewOpts struct {
	ShowComments   bool
	ShowSystemLogs bool
	OpenInBrowser  bool
	Web            bool
	OutputFormat   string

	CommentPageNumber int
	CommentLimit      int

	Notes []*gitlab.Note
	Issue *gitlab.Issue

	IO *iostreams.IOStreams
}

func NewCmdView(f *cmdutils.Factory, issueType issuable.IssueType) *cobra.Command {
	examplePath := "issues/123"

	if issueType == issuable.TypeIncident {
		examplePath = "issues/incident/123"
	}

	opts := &ViewOpts{
		IO: f.IO,
	}
	issueViewCmd := &cobra.Command{
		Use:     "view <id>",
		Short:   fmt.Sprintf(`Display the title, body, and other information about an %s.`, issueType),
		Long:    ``,
		Aliases: []string{"show"},
		Example: heredoc.Doc(fmt.Sprintf(`
			glab %[1]s view 123
			glab %[1]s show 123
			glab %[1]s view --web 123
			glab %[1]s view --comments 123
			glab %[1]s view https://gitlab.com/NAMESPACE/REPO/-/%s
		`, issueType, examplePath)),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			apiClient, err := f.HttpClient()
			if err != nil {
				return err
			}
			cfg, _ := f.Config()

			issue, baseRepo, err := issueutils.IssueFromArg(apiClient, f.BaseRepo, args[0])
			if err != nil {
				return err
			}

			opts.Issue = issue

			valid, msg := issuable.ValidateIncidentCmd(issueType, "view", opts.Issue)
			if !valid {
				fmt.Fprintln(opts.IO.StdErr, msg)
				return nil
			}

			// open in browser if --web flag is specified
			if opts.Web {
				if f.IO.IsaTTY && f.IO.IsErrTTY {
					fmt.Fprintf(opts.IO.StdErr, "Opening %s in your browser.\n", utils.DisplayURL(opts.Issue.WebURL))
				}

				browser, _ := cfg.Get(baseRepo.RepoHost(), "browser")
				return utils.OpenInBrowser(opts.Issue.WebURL, browser)
			}

			if opts.ShowComments {
				l := &gitlab.ListIssueNotesOptions{
					Sort: gitlab.Ptr("asc"),
				}
				if opts.CommentPageNumber != 0 {
					l.Page = opts.CommentPageNumber
				}
				if opts.CommentLimit != 0 {
					l.PerPage = opts.CommentLimit
				}
				opts.Notes, err = api.ListIssueNotes(apiClient, baseRepo.FullName(), opts.Issue.IID, l)
				if err != nil {
					return err
				}
			}

			glamourStyle, _ := cfg.Get(baseRepo.RepoHost(), "glamour_style")
			f.IO.ResolveBackgroundColor(glamourStyle)
			err = f.IO.StartPager()
			if err != nil {
				return err
			}
			defer f.IO.StopPager()
			if opts.OutputFormat == "json" {
				return printJSONIssue(opts)
			}
			if f.IO.IsErrTTY && f.IO.IsaTTY {
				return printTTYIssuePreview(opts)
			}
			return printRawIssuePreview(opts)
		},
	}

	issueViewCmd.Flags().BoolVarP(&opts.ShowComments, "comments", "c", false, fmt.Sprintf("Show %s comments and activities.", issueType))
	issueViewCmd.Flags().BoolVarP(&opts.ShowSystemLogs, "system-logs", "s", false, "Show system activities and logs.")
	issueViewCmd.Flags().BoolVarP(&opts.Web, "web", "w", false, fmt.Sprintf("Open %s in a browser. Uses the default browser, or the browser specified in the $BROWSER variable.", issueType))
	issueViewCmd.Flags().IntVarP(&opts.CommentPageNumber, "page", "p", 1, "Page number.")
	issueViewCmd.Flags().IntVarP(&opts.CommentLimit, "per-page", "P", 20, "Number of items to list per page.")
	issueViewCmd.Flags().StringVarP(&opts.OutputFormat, "output", "F", "text", "Format output as: text, json.")

	return issueViewCmd
}

func labelsList(opts *ViewOpts) string {
	return strings.Join(opts.Issue.Labels, ", ")
}

func assigneesList(opts *ViewOpts) string {
	assignees := utils.Map(opts.Issue.Assignees, func(a *gitlab.IssueAssignee) string {
		return a.Username
	})

	return strings.Join(assignees, ", ")
}

func issueState(opts *ViewOpts, c *iostreams.ColorPalette) (state string) {
	if opts.Issue.State == "opened" {
		state = c.Green("open")
	} else if opts.Issue.State == "locked" {
		state = c.Blue(opts.Issue.State)
	} else {
		state = c.Red(opts.Issue.State)
	}

	return
}

func printTTYIssuePreview(opts *ViewOpts) error {
	c := opts.IO.Color()
	issueTimeAgo := utils.TimeToPrettyTimeAgo(*opts.Issue.CreatedAt)
	// Header
	fmt.Fprint(opts.IO.StdOut, issueState(opts, c))
	fmt.Fprintf(opts.IO.StdOut, c.Gray(" • opened by %s %s\n"), opts.Issue.Author.Username, issueTimeAgo)
	fmt.Fprint(opts.IO.StdOut, c.Bold(opts.Issue.Title))
	fmt.Fprintf(opts.IO.StdOut, c.Gray(" #%d"), opts.Issue.IID)
	fmt.Fprintln(opts.IO.StdOut)

	// Description
	if opts.Issue.Description != "" {
		opts.Issue.Description, _ = utils.RenderMarkdown(opts.Issue.Description, opts.IO.BackgroundColor())
		fmt.Fprintln(opts.IO.StdOut, opts.Issue.Description)
	}

	fmt.Fprintf(opts.IO.StdOut, c.Gray("\n%d upvotes • %d downvotes • %d comments\n"), opts.Issue.Upvotes, opts.Issue.Downvotes, opts.Issue.UserNotesCount)

	// Meta information
	if labels := labelsList(opts); labels != "" {
		fmt.Fprint(opts.IO.StdOut, c.Bold("Labels: "))
		fmt.Fprintln(opts.IO.StdOut, labels)
	}
	if assignees := assigneesList(opts); assignees != "" {
		fmt.Fprint(opts.IO.StdOut, c.Bold("Assignees: "))
		fmt.Fprintln(opts.IO.StdOut, assignees)
	}
	if opts.Issue.Milestone != nil {
		fmt.Fprint(opts.IO.StdOut, c.Bold("Milestone: "))
		fmt.Fprintln(opts.IO.StdOut, opts.Issue.Milestone.Title)
	}
	if opts.Issue.State == "closed" {
		fmt.Fprintf(opts.IO.StdOut, "Closed by: %s %s\n", opts.Issue.ClosedBy.Username, issueTimeAgo)
	}

	// Comments
	if opts.ShowComments {
		fmt.Fprintln(opts.IO.StdOut, heredoc.Doc(`
			--------------------------------------------
			Comments / Notes
			--------------------------------------------
			`))
		if len(opts.Notes) > 0 {
			for _, note := range opts.Notes {
				if note.System && !opts.ShowSystemLogs {
					continue
				}
				createdAt := utils.TimeToPrettyTimeAgo(*note.CreatedAt)
				fmt.Fprint(opts.IO.StdOut, note.Author.Username)
				if note.System {
					fmt.Fprintf(opts.IO.StdOut, " %s ", note.Body)
					fmt.Fprintln(opts.IO.StdOut, c.Gray(createdAt))
				} else {
					body, _ := utils.RenderMarkdown(note.Body, opts.IO.BackgroundColor())
					fmt.Fprint(opts.IO.StdOut, " commented ")
					fmt.Fprintf(opts.IO.StdOut, c.Gray("%s\n"), createdAt)
					fmt.Fprintln(opts.IO.StdOut, utils.Indent(body, " "))
				}
				fmt.Fprintln(opts.IO.StdOut)
			}
		} else {
			fmt.Fprintf(opts.IO.StdOut, "There are no comments on this %s.\n", *opts.Issue.IssueType)
		}
	}

	fmt.Fprintf(opts.IO.StdOut, c.Gray("\nView this %s on GitLab: %s\n"), *opts.Issue.IssueType, opts.Issue.WebURL)

	return nil
}

func printRawIssuePreview(opts *ViewOpts) error {
	fmt.Fprint(opts.IO.StdOut, rawIssuePreview(opts))

	return nil
}

func rawIssuePreview(opts *ViewOpts) string {
	var out string

	assignees := assigneesList(opts)
	labels := labelsList(opts)

	out += fmt.Sprintf("title:\t%s\n", opts.Issue.Title)
	out += fmt.Sprintf("state:\t%s\n", issueState(opts, opts.IO.Color()))
	out += fmt.Sprintf("author:\t%s\n", opts.Issue.Author.Username)
	out += fmt.Sprintf("labels:\t%s\n", labels)
	out += fmt.Sprintf("comments:\t%d\n", opts.Issue.UserNotesCount)
	out += fmt.Sprintf("assignees:\t%s\n", assignees)
	if opts.Issue.Milestone != nil {
		out += fmt.Sprintf("milestone:\t%s\n", opts.Issue.Milestone.Title)
	}

	out += "--\n"
	out += fmt.Sprintf("%s\n", opts.Issue.Description)

	out += RawIssuableNotes(opts.Notes, opts.ShowComments, opts.ShowSystemLogs, *opts.Issue.IssueType)

	return out
}

// RawIssuableNotes returns a list of comments/notes in a raw format
func RawIssuableNotes(notes []*gitlab.Note, showComments bool, showSystemLogs bool, issuableName string) string {
	var out string

	if showComments {
		out += "\n--\ncomments/notes:\n\n"

		if len(notes) > 0 {
			for _, note := range notes {
				if note.System && !showSystemLogs {
					continue
				}

				if note.System {
					out += fmt.Sprintf("%s %s %s\n\n", note.Author.Username, note.Body, note.CreatedAt.String())
				} else {
					out += fmt.Sprintf("%s commented %s\n%s\n\n", note.Author.Username, note.CreatedAt.String(), note.Body)
				}
			}
		} else {
			out += fmt.Sprintf("There are no comments on this %s.\n", issuableName)
		}
	}

	return out
}

func printJSONIssue(opts *ViewOpts) error {
	// var notes []gitlab.Note
	if opts.ShowComments {

		extendedIssue := IssueWithNotes{opts.Issue, opts.Notes}
		issueJSON, _ := json.Marshal(extendedIssue)
		fmt.Fprintln(opts.IO.StdOut, string(issueJSON))
	} else {
		issueJSON, _ := json.Marshal(opts.Issue)
		fmt.Fprintln(opts.IO.StdOut, string(issueJSON))
	}
	return nil
}
