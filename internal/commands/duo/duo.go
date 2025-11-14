package duo

import (
	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"

	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	duoAskCmd "gitlab.com/gitlab-org/cli/internal/commands/duo/ask"
)

func NewCmdDuo(f cmdutils.Factory) *cobra.Command {
	duoCmd := &cobra.Command{
		Use:   "duo <command> prompt",
		Short: "Work with GitLab Duo",
		Long: heredoc.Doc(`
			Work with GitLab Duo, our AI-native assistant for the command line.

			GitLab Duo for the CLI integrates AI capabilities directly into your terminal
			workflow. It helps you retrieve forgotten Git commands and offers guidance on
			Git operations. You can accomplish specific tasks without switching contexts.
		`),
	}

	duoCmd.AddCommand(duoAskCmd.NewCmdAsk(f))

	return duoCmd
}
