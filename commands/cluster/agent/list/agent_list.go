package list

import (
	"fmt"

	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/commands/cluster/agent/agentutils"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/pkg/utils"

	"github.com/spf13/cobra"
	"github.com/xanzy/go-gitlab"
)

var factory *cmdutils.Factory

func NewCmdAgentList(f *cmdutils.Factory) *cobra.Command {
	factory = f
	agentListCmd := &cobra.Command{
		Use:     "list [flags]",
		Short:   `List GitLab Agents for Kubernetes in a project.`,
		Long:    ``,
		Aliases: []string{"ls"},
		Args:    cobra.MaximumNArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			factory = f
			page, err := cmd.Flags().GetUint("page")
			if err != nil {
				return err
			}
			perPage, err := cmd.Flags().GetUint("per-page")
			if err != nil {
				return err
			}
			return listAgents(int(page), int(perPage))
		},
	}
	agentListCmd.Flags().UintP("page", "p", 1, "Page number.")
	agentListCmd.Flags().UintP("per-page", "P", uint(api.DefaultListLimit), "Number of items to list per page.")

	return agentListCmd
}

func listAgents(page, perPage int) error {
	apiClient, err := factory.HttpClient()
	if err != nil {
		return err
	}

	repo, err := factory.BaseRepo()
	if err != nil {
		return err
	}

	agents, err := api.ListAgents(apiClient, repo.FullName(), &gitlab.ListAgentsOptions{
		Page:    page,
		PerPage: perPage,
	})
	if err != nil {
		return err
	}

	title := utils.NewListTitle("agent")
	title.RepoName = repo.FullName()
	title.Page = page
	title.CurrentPageTotal = len(agents)
	err = factory.IO.StartPager()
	if err != nil {
		return err
	}
	defer factory.IO.StopPager()

	fmt.Fprintf(factory.IO.StdOut, "%s\n%s\n", title.Describe(), agentutils.DisplayAllAgents(factory.IO, agents))
	return nil
}
