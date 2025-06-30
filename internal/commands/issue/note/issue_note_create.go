package note

import (
	"github.com/spf13/cobra"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/commands/issuable"

	issuableNoteCmd "gitlab.com/gitlab-org/cli/internal/commands/issuable/note"
)

func NewCmdNote(f cmdutils.Factory) *cobra.Command {
	return issuableNoteCmd.NewCmdNote(f, issuable.TypeIssue)
}
