package cancel

import (
	"github.com/spf13/cobra"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	cancelJobCmd "gitlab.com/gitlab-org/cli/internal/commands/ci/cancel/job"
	cancelPipelineCmd "gitlab.com/gitlab-org/cli/internal/commands/ci/cancel/pipeline"
)

func NewCmdCancel(f cmdutils.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cancel <command>",
		Short: "Cancel a running pipeline or job.",
	}

	cmd.AddCommand(cancelPipelineCmd.NewCmdCancel(f))
	cmd.AddCommand(cancelJobCmd.NewCmdCancel(f))

	return cmd
}
