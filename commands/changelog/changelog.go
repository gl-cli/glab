package changelog

import (
	"github.com/spf13/cobra"
	changelogGenerateCmd "gitlab.com/gitlab-org/cli/commands/changelog/generate"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
)

func NewCmdChangelog(f *cmdutils.Factory) *cobra.Command {
	changelogCmd := &cobra.Command{
		Use:   "changelog <command> [flags]",
		Short: `Interact with the changelog API.`,
		Long:  ``,
	}

	// Subcommands
	changelogCmd.AddCommand(changelogGenerateCmd.NewCmdGenerate(f))

	return changelogCmd
}
