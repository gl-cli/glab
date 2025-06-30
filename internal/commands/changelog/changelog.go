package changelog

import (
	"github.com/spf13/cobra"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	changelogGenerateCmd "gitlab.com/gitlab-org/cli/internal/commands/changelog/generate"
)

func NewCmdChangelog(f cmdutils.Factory) *cobra.Command {
	changelogCmd := &cobra.Command{
		Use:   "changelog <command> [flags]",
		Short: `Interact with the changelog API.`,
		Long:  ``,
	}

	// Subcommands
	changelogCmd.AddCommand(changelogGenerateCmd.NewCmdGenerate(f))

	return changelogCmd
}
