package subscribe

import (
	"errors"
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/commands/issuable"
	"gitlab.com/gitlab-org/cli/commands/issue/issueutils"
)

var subscribingMessage = map[issuable.IssueType]string{
	issuable.TypeIssue:    "Subscribing to issue",
	issuable.TypeIncident: "Subscribing to incident",
}

func NewCmdSubscribe(f *cmdutils.Factory, issueType issuable.IssueType) *cobra.Command {
	examplePath := "issues/123"

	if issueType == issuable.TypeIncident {
		examplePath = "issues/incident/123"
	}

	issueSubscribeCmd := &cobra.Command{
		Use:     "subscribe <id>",
		Short:   fmt.Sprintf(`Subscribe to an %s.`, issueType),
		Long:    ``,
		Aliases: []string{"sub"},
		Example: heredoc.Doc(fmt.Sprintf(`
			glab %[1]s subscribe 123
			glab %[1]s sub 123
			glab %[1]s subscribe https://gitlab.com/OWNER/REPO/-/%[2]s
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
				valid, msg := issuable.ValidateIncidentCmd(issueType, "subscribe", issue)
				if !valid {
					fmt.Fprintln(f.IO.StdOut, msg)
					continue
				}

				if f.IO.IsaTTY && f.IO.IsErrTTY {
					fmt.Fprintf(
						f.IO.StdOut,
						"- %s #%d in %s\n",
						subscribingMessage[issueType],
						issue.IID,
						c.Cyan(repo.FullName()),
					)
				}

				issue, err := api.SubscribeToIssue(apiClient, repo.FullName(), issue.IID, nil)
				if err != nil {
					if errors.Is(err, api.ErrIssuableUserAlreadySubscribed) {
						fmt.Fprintf(
							f.IO.StdOut,
							"%s You are already subscribed to this %s.\n\n",
							c.FailedIcon(),
							issueType,
						)
						return nil // the error already handled
					}
					return err
				}

				fmt.Fprintln(f.IO.StdOut, c.GreenCheck(), "Subscribed")
				fmt.Fprintln(f.IO.StdOut, issueutils.DisplayIssue(c, issue, f.IO.IsaTTY))
			}
			return nil
		},
	}

	return issueSubscribeCmd
}
