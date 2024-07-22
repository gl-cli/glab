package close

import (
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"gitlab.com/gitlab-org/cli/api"

	"github.com/spf13/cobra"
	"github.com/xanzy/go-gitlab"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/commands/issuable"
	"gitlab.com/gitlab-org/cli/commands/issue/issueutils"
)

var closingMessage = map[issuable.IssueType]string{
	issuable.TypeIssue:    "Closing issue",
	issuable.TypeIncident: "Resolving incident",
}

var closedMessage = map[issuable.IssueType]string{
	issuable.TypeIssue:    "Closed issue",
	issuable.TypeIncident: "Resolved incident",
}

func NewCmdClose(f *cmdutils.Factory, issueType issuable.IssueType) *cobra.Command {
	examplePath := "issues/123"
	aliases := []string{}

	if issueType == issuable.TypeIncident {
		examplePath = "issues/incident/123"
		aliases = []string{"resolve"}
	}

	issueCloseCmd := &cobra.Command{
		Use:     "close [<id> | <url>] [flags]",
		Short:   fmt.Sprintf(`Close an %s.`, issueType),
		Long:    ``,
		Aliases: aliases,
		Example: heredoc.Doc(fmt.Sprintf(`
			glab %[1]s close 123
			glab %[1]s close https://gitlab.com/NAMESPACE/REPO/-/%s
		`, issueType, examplePath)),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			apiClient, err := f.HttpClient()
			if err != nil {
				return err
			}

			issues, repo, err := issueutils.IssuesFromArgs(apiClient, f.BaseRepo, args)
			if err != nil {
				return err
			}

			l := &gitlab.UpdateIssueOptions{}
			l.StateEvent = gitlab.Ptr("close")

			c := f.IO.Color()

			for _, issue := range issues {
				valid, msg := issuable.ValidateIncidentCmd(issueType, "close", issue)
				if !valid {
					fmt.Fprintln(f.IO.StdOut, msg)
					continue
				}

				fmt.Fprintf(f.IO.StdOut, "- %s...\n", closingMessage[issueType])
				issue, err := api.UpdateIssue(apiClient, repo.FullName(), issue.IID, l)
				if err != nil {
					return err
				}
				fmt.Fprintf(f.IO.StdOut, "%s %s #%d\n", c.RedCheck(), closedMessage[issueType], issue.IID)
				fmt.Fprintln(f.IO.StdOut, issueutils.DisplayIssue(c, issue, f.IO.IsaTTY))
			}
			return nil
		},
	}
	return issueCloseCmd
}
