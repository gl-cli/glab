package ask

import (
	"gitlab.com/gitlab-org/cli/commands/ask/git"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"

	"github.com/spf13/cobra"
)

func NewCmd(f *cmdutils.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ask <command> prompt",
		Short: "Generate terminal commands from natural language. (Experimental.)",
		Long:  ``,
	}

	cmd.AddCommand(git.NewCmd(f))

	return cmd
}
