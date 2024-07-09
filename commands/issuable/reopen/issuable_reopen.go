package reopen

import (
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/commands/issuable"
	"gitlab.com/gitlab-org/cli/commands/issue/issueutils"

	"github.com/spf13/cobra"
	"github.com/xanzy/go-gitlab"
)

var (
	description = map[issuable.IssueType]string{
		issuable.TypeIssue:    "Reopen a closed issue.",
		issuable.TypeIncident: "Reopen a resolved incident.",
	}

	reopeningMessage = map[issuable.IssueType]string{
		issuable.TypeIssue:    "Reopening issue",
		issuable.TypeIncident: "Reopening incident",
	}

	reopenedMessage = map[issuable.IssueType]string{
		issuable.TypeIssue:    "Reopened issue",
		issuable.TypeIncident: "Reopened incident",
	}
)

func NewCmdReopen(f *cmdutils.Factory, issueType issuable.IssueType) *cobra.Command {
	examplePath := "issues/123"

	if issueType == issuable.TypeIncident {
		examplePath = "issues/incident/123"
	}

	issueReopenCmd := &cobra.Command{
		Use:     "reopen [<id> | <url>] [flags]",
		Short:   description[issueType],
		Long:    ``,
		Aliases: []string{"open"},
		Example: heredoc.Doc(fmt.Sprintf(`
			glab %[1]s reopen 123
			glab %[1]s open 123
			glab %[1]s reopen https://gitlab.com/NAMESPACE/REPO/-/%s
		`, issueType, examplePath)),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			out := f.IO.StdOut
			c := f.IO.Color()

			apiClient, err := f.HttpClient()
			if err != nil {
				return err
			}

			issues, repo, err := issueutils.IssuesFromArgs(apiClient, f.BaseRepo, args)
			if err != nil {
				return err
			}

			l := &gitlab.UpdateIssueOptions{}
			l.StateEvent = gitlab.Ptr("reopen")

			for _, issue := range issues {
				valid, msg := issuable.ValidateIncidentCmd(issueType, "reopen", issue)
				if !valid {
					fmt.Fprintln(f.IO.StdOut, msg)
					continue
				}

				fmt.Fprintf(out, "- %s...\n", reopeningMessage[issueType])
				issue, err := api.UpdateIssue(apiClient, repo.FullName(), issue.IID, l)
				if err != nil {
					return err
				}

				fmt.Fprintf(out, "%s %s #%d.\n", c.GreenCheck(), reopenedMessage[issueType], issue.IID)
				fmt.Fprintln(out, issueutils.DisplayIssue(c, issue, f.IO.IsaTTY))
			}
			return nil
		},
	}

	return issueReopenCmd
}
