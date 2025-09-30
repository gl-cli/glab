package duo

import (
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	duoAskCmd "gitlab.com/gitlab-org/cli/internal/commands/duo/ask"
	"gitlab.com/gitlab-org/cli/internal/commands/duo/claude"
	"gitlab.com/gitlab-org/cli/internal/commands/duo/codex"

	"github.com/spf13/cobra"
)

func NewCmdDuo(f cmdutils.Factory) *cobra.Command {
	duoCmd := &cobra.Command{
		Use:   "duo <command> prompt",
		Short: "Work with GitLab Duo",
		Long:  ``,
	}

	duoCmd.AddCommand(duoAskCmd.NewCmdAsk(f))
	duoCmd.AddCommand(claude.NewCmdClaude(f))
	duoCmd.AddCommand(codex.NewCmdCodex(f))

	return duoCmd
}
