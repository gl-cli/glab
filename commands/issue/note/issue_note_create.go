package note

import (
	"github.com/spf13/cobra"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/commands/issuable"

	issuableNoteCmd "gitlab.com/gitlab-org/cli/commands/issuable/note"
)

func NewCmdNote(f *cmdutils.Factory) *cobra.Command {
	return issuableNoteCmd.NewCmdNote(f, issuable.TypeIssue)
}
