package duo

import (
	"fmt"
	"os"

	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	duoAskCmd "gitlab.com/gitlab-org/cli/commands/duo/ask"

	"github.com/spf13/cobra"
)

func NewCmdDuo(f *cmdutils.Factory) *cobra.Command {
	duoCmd := &cobra.Command{
		Use:     "duo <command> prompt",
		Short:   "Generate terminal commands from natural language. (Experimental.)",
		Long:    ``,
		Aliases: []string{"ask"},
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintf(os.Stderr, "Aliases 'ask' is deprecated. Please use 'duo' instead.\n\n")
			_ = cmd.Help()
		},
	}

	duoCmd.AddCommand(duoAskCmd.NewCmdAsk(f))

	return duoCmd
}
