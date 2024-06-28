package cluster

import (
	"github.com/spf13/cobra"
	agentCmd "gitlab.com/gitlab-org/cli/commands/cluster/agent"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
)

func NewCmdCluster(f *cmdutils.Factory) *cobra.Command {
	clusterCmd := &cobra.Command{
		Use:   "cluster <command> [flags]",
		Short: `Manage GitLab Agents for Kubernetes and their clusters.`,
		Long:  ``,
	}

	cmdutils.EnableRepoOverride(clusterCmd, f)

	clusterCmd.AddCommand(agentCmd.NewCmdAgent(f))

	return clusterCmd
}
