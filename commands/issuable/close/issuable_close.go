package close

import (
	"fmt"

	"github.com/MakeNowJust/heredoc"
	"gitlab.com/gitlab-org/cli/api"

	"github.com/spf13/cobra"
	"github.com/xanzy/go-gitlab"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/commands/issuable"
	"gitlab.com/gitlab-org/cli/commands/issue/issueutils"
)

var closingMessage = map[issuable.IssueType]string{
	issuable.TypeIssue:    "Closing Issue",
	issuable.TypeIncident: "Resolving Incident",
}

var closedMessage = map[issuable.IssueType]string{
	issuable.TypeIssue:    "Closed Issue",
	issuable.TypeIncident: "Resolved Incident",
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
		Short:   fmt.Sprintf(`Close an %s`, issueType),
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
			l.StateEvent = gitlab.String("close")

			c := f.IO.Color()

			for _, issue := range issues {
				// Issues and incidents are the same kind, but with different issueType.
				// `issue close` can close issues of all types including incidents
				// `incident close` on the other hand, should close only incidents, and treat all other issue types as not found
				//
				// When using `incident close` with non incident's IDs, print an error.
				if issueType == issuable.TypeIncident && *issue.IssueType != string(issuable.TypeIncident) {
					fmt.Fprintln(f.IO.StdOut, "Incident not found, but an issue with the provided ID exists. Run `glab issue close <id>` to close it.")
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
