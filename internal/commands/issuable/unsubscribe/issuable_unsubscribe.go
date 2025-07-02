package unsubscribe

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/MakeNowJust/heredoc/v2"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/commands/issuable"
	"gitlab.com/gitlab-org/cli/internal/commands/issue/issueutils"

	"github.com/spf13/cobra"
)

// errIssuableUserNotSubscribed received when trying to unsubscribe from an issue the user is not subscribed to
var errIssuableUserNotSubscribed = errors.New("you are not subscribed to this issue")

var unsubscribingMessage = map[issuable.IssueType]string{
	issuable.TypeIssue:    "Unsubscribing from issue",
	issuable.TypeIncident: "Unsubscribing from incident",
}

func NewCmdUnsubscribe(f cmdutils.Factory, issueType issuable.IssueType) *cobra.Command {
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
			$ glab %[1]s unsubscribe 123
			$ glab %[1]s unsub 123
			$ glab %[1]s unsubscribe https://gitlab.com/OWNER/REPO/-/%[2]s
		`, issueType, examplePath)),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := f.IO().Color()
			gitlabClient, err := f.HttpClient()
			if err != nil {
				return err
			}

			issues, repo, err := issueutils.IssuesFromArgs(f.ApiClient, gitlabClient, f.BaseRepo, f.DefaultHostname(), args)
			if err != nil {
				return err
			}

			for _, issue := range issues {
				valid, msg := issuable.ValidateIncidentCmd(issueType, "unsubscribe", issue)
				if !valid {
					fmt.Fprintln(f.IO().StdOut, msg)
					continue
				}

				if f.IO().IsaTTY && f.IO().IsErrTTY {
					fmt.Fprintf(
						f.IO().StdOut,
						"- %s #%d in %s\n",
						unsubscribingMessage[issueType],
						issue.IID,
						c.Cyan(repo.FullName()),
					)
				}

				issue, err := unsubscribe(gitlabClient, repo.FullName(), issue.IID, nil)
				if err != nil {
					if errors.Is(err, errIssuableUserNotSubscribed) {
						fmt.Fprintf(
							f.IO().StdOut,
							"%s You are not subscribed to this %s.\n\n",
							c.FailedIcon(),
							issueType,
						)
						return nil // the error already handled
					}
					return err
				}

				fmt.Fprintln(f.IO().StdOut, c.GreenCheck(), "Unsubscribed")
				fmt.Fprintln(f.IO().StdOut, issueutils.DisplayIssue(c, issue, f.IO().IsaTTY))
			}
			return nil
		},
	}

	return issueUnsubscribeCmd
}

func unsubscribe(client *gitlab.Client, projectID any, issueID int, opts gitlab.RequestOptionFunc) (*gitlab.Issue, error) {
	issue, resp, err := client.Issues.UnsubscribeFromIssue(projectID, issueID, opts)
	if err != nil {
		if resp != nil {
			// If the user is not subscribed to the issue, the status code 304 is returned.
			if resp.StatusCode == http.StatusNotModified {
				return nil, errIssuableUserNotSubscribed
			}
		}
		return issue, err
	}

	return issue, nil
}
