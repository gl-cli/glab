package cancel

import (
	"github.com/spf13/cobra"
	cancelJobCmd "gitlab.com/gitlab-org/cli/commands/ci/cancel/job"
	cancelPipelineCmd "gitlab.com/gitlab-org/cli/commands/ci/cancel/pipeline"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
)

func NewCmdCancel(f *cmdutils.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cancel <command>",
		Short: "Cancel a running pipeline or job.",
	}

	cmd.AddCommand(cancelPipelineCmd.NewCmdCancel(f))
	cmd.AddCommand(cancelJobCmd.NewCmdCancel(f))

	return cmd
}
