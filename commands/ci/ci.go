package ci

import (
	jobArtifactCmd "gitlab.com/gitlab-org/cli/commands/ci/artifact"
	ciConfigCmd "gitlab.com/gitlab-org/cli/commands/ci/config"
	pipeDeleteCmd "gitlab.com/gitlab-org/cli/commands/ci/delete"
	pipeGetCmd "gitlab.com/gitlab-org/cli/commands/ci/get"
	legacyCICmd "gitlab.com/gitlab-org/cli/commands/ci/legacyci"
	ciLintCmd "gitlab.com/gitlab-org/cli/commands/ci/lint"
	pipeListCmd "gitlab.com/gitlab-org/cli/commands/ci/list"
	pipeRetryCmd "gitlab.com/gitlab-org/cli/commands/ci/retry"
	pipeRunCmd "gitlab.com/gitlab-org/cli/commands/ci/run"
	pipeRunTrigCmd "gitlab.com/gitlab-org/cli/commands/ci/run_trig"
	pipeStatusCmd "gitlab.com/gitlab-org/cli/commands/ci/status"
	ciTraceCmd "gitlab.com/gitlab-org/cli/commands/ci/trace"
	jobPlayCmd "gitlab.com/gitlab-org/cli/commands/ci/trigger"
	ciViewCmd "gitlab.com/gitlab-org/cli/commands/ci/view"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"

	"github.com/spf13/cobra"
)

func NewCmdCI(f *cmdutils.Factory) *cobra.Command {
	ciCmd := &cobra.Command{
		Use:     "ci <command> [flags]",
		Short:   `Work with GitLab CI/CD pipelines and jobs`,
		Long:    ``,
		Aliases: []string{"pipe", "pipeline"},
	}

	cmdutils.EnableRepoOverride(ciCmd, f)

	ciCmd.AddCommand(legacyCICmd.NewCmdCI(f))
	ciCmd.AddCommand(ciTraceCmd.NewCmdTrace(f))
	ciCmd.AddCommand(ciViewCmd.NewCmdView(f))
	ciCmd.AddCommand(ciLintCmd.NewCmdLint(f))
	ciCmd.AddCommand(pipeDeleteCmd.NewCmdDelete(f))
	ciCmd.AddCommand(pipeListCmd.NewCmdList(f))
	ciCmd.AddCommand(pipeStatusCmd.NewCmdStatus(f))
	ciCmd.AddCommand(pipeRetryCmd.NewCmdRetry(f))
	ciCmd.AddCommand(pipeRunCmd.NewCmdRun(f))
	ciCmd.AddCommand(jobPlayCmd.NewCmdTrigger(f))
	ciCmd.AddCommand(pipeRunTrigCmd.NewCmdRunTrig(f))
	ciCmd.AddCommand(jobArtifactCmd.NewCmdRun(f))
	ciCmd.AddCommand(pipeGetCmd.NewCmdGet(f))
	ciCmd.AddCommand(ciConfigCmd.NewCmdConfig(f))
	return ciCmd
}
