package unsubscribe

import (
	"errors"
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/commands/issuable"
	"gitlab.com/gitlab-org/cli/commands/issue/issueutils"

	"github.com/spf13/cobra"
)

var unsubscribingMessage = map[issuable.IssueType]string{
	issuable.TypeIssue:    "Unsubscribing from issue",
	issuable.TypeIncident: "Unsubscribing from incident",
}

func NewCmdUnsubscribe(f *cmdutils.Factory, issueType issuable.IssueType) *cobra.Command {
	examplePath := "issues/123"

	if issueType == issuable.TypeIncident {
		examplePath = "issues/incident/123"
	}

	issueUnsubscribeCmd := &cobra.Command{
		Use:     "unsubscribe <id>",
		Short:   fmt.Sprintf(`Unsubscribe from an %s.`, issueType),
		Long:    ``,
		Aliases: []string{"unsub"},
		Example: heredoc.Doc(fmt.Sprintf(`
			glab %[1]s unsubscribe 123
			glab %[1]s unsub 123
			glab %[1]s unsubscribe https://gitlab.com/OWNER/REPO/-/%[2]s
		`, issueType, examplePath)),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := f.IO.Color()
			apiClient, err := f.HttpClient()
			if err != nil {
				return err
			}

			issues, repo, err := issueutils.IssuesFromArgs(apiClient, f.BaseRepo, args)
			if err != nil {
				return err
			}

			for _, issue := range issues {
				valid, msg := issuable.ValidateIncidentCmd(issueType, "unsubscribe", issue)
				if !valid {
					fmt.Fprintln(f.IO.StdOut, msg)
					continue
				}

				if f.IO.IsaTTY && f.IO.IsErrTTY {
					fmt.Fprintf(
						f.IO.StdOut,
						"- %s #%d in %s\n",
						unsubscribingMessage[issueType],
						issue.IID,
						c.Cyan(repo.FullName()),
					)
				}

				issue, err := api.UnsubscribeFromIssue(apiClient, repo.FullName(), issue.IID, nil)
				if err != nil {
					if errors.Is(err, api.ErrIssuableUserNotSubscribed) {
						fmt.Fprintf(
							f.IO.StdOut,
							"%s You are not subscribed to this %s.\n\n",
							c.FailedIcon(),
							issueType,
						)
						return nil // the error already handled
					}
					return err
				}

				fmt.Fprintln(f.IO.StdOut, c.GreenCheck(), "Unsubscribed")
				fmt.Fprintln(f.IO.StdOut, issueutils.DisplayIssue(c, issue, f.IO.IsaTTY))
			}
			return nil
		},
	}

	return issueUnsubscribeCmd
}
