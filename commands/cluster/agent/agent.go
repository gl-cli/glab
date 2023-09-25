package cluster

import (
	"github.com/spf13/cobra"
	agentListCmd "gitlab.com/gitlab-org/cli/commands/cluster/agent/list"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
)

func NewCmdAgent(f *cmdutils.Factory) *cobra.Command {
	agentCmd := &cobra.Command{
		Use:   "agent <command> [flags]",
		Short: `Manage GitLab Agents for Kubernetes`,
		Long:  ``,
	}

	cmdutils.EnableRepoOverride(agentCmd, f)

	agentCmd.AddCommand(agentListCmd.NewCmdAgentList(f))

	return agentCmd
}
