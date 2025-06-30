package iteration

import (
	"github.com/spf13/cobra"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	iterationListCmd "gitlab.com/gitlab-org/cli/internal/commands/iteration/list"
)

func NewCmdIteration(f cmdutils.Factory) *cobra.Command {
	iterationCmd := &cobra.Command{
		Use:   "iteration <command> [flags]",
		Short: `Retrieve iteration information.`,
		Long:  ``,
	}

	cmdutils.EnableRepoOverride(iterationCmd, f)

	iterationCmd.AddCommand(iterationListCmd.NewCmdList(f))
	return iterationCmd
}
