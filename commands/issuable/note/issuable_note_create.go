package note

import (
	"errors"
	"fmt"
	"strings"

	"gitlab.com/gitlab-org/cli/commands/issuable"
	"gitlab.com/gitlab-org/cli/commands/issue/issueutils"

	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/pkg/utils"

	"github.com/spf13/cobra"
	gitlab "github.com/xanzy/go-gitlab"
)

func NewCmdNote(f *cmdutils.Factory, issueType issuable.IssueType) *cobra.Command {
	issueNoteCreateCmd := &cobra.Command{
		Use:     fmt.Sprintf("note <%s-id>", issueType),
		Aliases: []string{"comment"},
		Short:   fmt.Sprintf("Comment on an %s in GitLab.", issueType),
		Long:    ``,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			out := f.IO.StdOut

			apiClient, err := f.HttpClient()
			if err != nil {
				return err
			}

			issue, repo, err := issueutils.IssueFromArg(apiClient, f.BaseRepo, args[0])
			if err != nil {
				return err
			}

			valid, msg := issuable.ValidateIncidentCmd(issueType, "comment", issue)
			if !valid {
				fmt.Fprintln(f.IO.StdOut, msg)
				return nil
			}

			body, _ := cmd.Flags().GetString("message")

			if strings.TrimSpace(body) == "" {
				editor, err := cmdutils.GetEditor(f.Config)
				if err != nil {
					return err
				}

				body = utils.Editor(utils.EditorOptions{
					Label:         "Message:",
					Help:          "Enter the note's message. ",
					FileName:      "ISSUE_NOTE_EDITMSG",
					EditorCommand: editor,
				})
			}

			if strings.TrimSpace(body) == "" {
				return errors.New("aborted... Note is empty.")
			}

			noteInfo, err := api.CreateIssueNote(apiClient, repo.FullName(), issue.IID, &gitlab.CreateIssueNoteOptions{
				Body: &body,
			})
			if err != nil {
				return err
			}

			fmt.Fprintf(out, "%s#note_%d\n", issue.WebURL, noteInfo.ID)
			return nil
		},
	}
	issueNoteCreateCmd.Flags().StringP("message", "m", "", "Message text.")

	return issueNoteCreateCmd
}
