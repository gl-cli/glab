package note

import (
	"fmt"

	"gitlab.com/gitlab-org/cli/commands/mr/mrutils"

	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/pkg/utils"

	"github.com/spf13/cobra"
	"github.com/xanzy/go-gitlab"
)

func NewCmdNote(f *cmdutils.Factory) *cobra.Command {
	mrCreateNoteCmd := &cobra.Command{
		Use:     "note [<id> | <branch>]",
		Aliases: []string{"comment"},
		Short:   "Add a comment or note to a merge request.",
		Long:    ``,
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			apiClient, err := f.HttpClient()
			if err != nil {
				return err
			}

			mr, repo, err := mrutils.MRFromArgs(f, args, "any")
			if err != nil {
				return err
			}

			body, _ := cmd.Flags().GetString("message")

			if body == "" {
				editor, err := cmdutils.GetEditor(f.Config)
				if err != nil {
					return err
				}

				body = utils.Editor(utils.EditorOptions{
					Label:         "Note message:",
					Help:          "Enter the note message for the merge request. ",
					FileName:      "*_MR_NOTE_EDITMSG.md",
					EditorCommand: editor,
				})
			}
			if body == "" {
				return fmt.Errorf("aborted... Note has an empty message.")
			}

			uniqueNoteEnabled, _ := cmd.Flags().GetBool("unique")

			if uniqueNoteEnabled {
				opts := &gitlab.ListMergeRequestNotesOptions{}
				notes, err := api.ListMRNotes(apiClient, repo.FullName(), mr.IID, opts)
				if err != nil {
					return fmt.Errorf("running merge request note deduplication: %v", err)
				}
				for _, noteInfo := range notes {
					if noteInfo.Body == body {
						fmt.Fprintf(f.IO.StdOut, "%s#note_%d\n", mr.WebURL, noteInfo.ID)
						return nil
					}
				}
			}

			noteInfo, err := api.CreateMRNote(apiClient, repo.FullName(), mr.IID, &gitlab.CreateMergeRequestNoteOptions{
				Body: &body,
			})
			if err != nil {
				return err
			}

			fmt.Fprintf(f.IO.StdOut, "%s#note_%d\n", mr.WebURL, noteInfo.ID)
			return nil
		},
	}

	mrCreateNoteCmd.Flags().StringP("message", "m", "", "Comment or note message.")
	mrCreateNoteCmd.Flags().Bool("unique", false, "Don't create a comment or note if it already exists.")
	return mrCreateNoteCmd
}
