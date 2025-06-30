package agent

import (
	"github.com/spf13/cobra"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	agentBootstrapCmd "gitlab.com/gitlab-org/cli/internal/commands/cluster/agent/bootstrap"
	checkManifestUsageCmd "gitlab.com/gitlab-org/cli/internal/commands/cluster/agent/check_manifest_usage"
	agentGetTokenCmd "gitlab.com/gitlab-org/cli/internal/commands/cluster/agent/get_token"
	agentListCmd "gitlab.com/gitlab-org/cli/internal/commands/cluster/agent/list"
	agentUpdateKubeconfigCmd "gitlab.com/gitlab-org/cli/internal/commands/cluster/agent/update_kubeconfig"
)

func NewCmdAgent(f cmdutils.Factory) *cobra.Command {
	agentCmd := &cobra.Command{
		Use:   "agent <command> [flags]",
		Short: `Manage GitLab Agents for Kubernetes.`,
		Long:  ``,
	}

	cmdutils.EnableRepoOverride(agentCmd, f)

	agentCmd.AddCommand(agentListCmd.NewCmdAgentList(f))
	agentCmd.AddCommand(agentGetTokenCmd.NewCmdAgentGetToken(f))
	agentCmd.AddCommand(agentUpdateKubeconfigCmd.NewCmdAgentUpdateKubeconfig(f))
	agentCmd.AddCommand(checkManifestUsageCmd.NewCmdCheckManifestUsage(f))

	agentCmd.AddCommand(agentBootstrapCmd.NewCmdAgentBootstrap(
		f,
		agentBootstrapCmd.EnsureRequirements,
		agentBootstrapCmd.NewAPI,
		agentBootstrapCmd.NewLocalKubectlWrapper,
		agentBootstrapCmd.NewLocalFluxWrapper,
		agentBootstrapCmd.NewCmd,
	))

	return agentCmd
}
