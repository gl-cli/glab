package list

import (
	"gitlab.com/gitlab-org/cli/internal/api"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/commands/cluster/agent/agentutils"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/internal/iostreams"
	"gitlab.com/gitlab-org/cli/internal/utils"

	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

type options struct {
	httpClient func() (*gitlab.Client, error)
	io         *iostreams.IOStreams
	baseRepo   func() (glrepo.Interface, error)

	page, perPage uint
}

func NewCmdAgentList(f cmdutils.Factory) *cobra.Command {
	opts := options{
		io:         f.IO(),
		httpClient: f.HttpClient,
		baseRepo:   f.BaseRepo,
	}
	agentListCmd := &cobra.Command{
		Use:     "list [flags]",
		Short:   `List GitLab Agents for Kubernetes in a project.`,
		Long:    ``,
		Aliases: []string{"ls"},
		Args:    cobra.MaximumNArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			return opts.run()
		},
	}
	agentListCmd.Flags().UintVarP(&opts.page, "page", "p", 1, "Page number.")
	agentListCmd.Flags().UintVarP(&opts.perPage, "per-page", "P", uint(api.DefaultListLimit), "Number of items to list per page.")

	return agentListCmd
}

func (o *options) run() error {
	apiClient, err := o.httpClient()
	if err != nil {
		return err
	}

	repo, err := o.baseRepo()
	if err != nil {
		return err
	}

	agents, _, err := apiClient.ClusterAgents.ListAgents(repo.FullName(), &gitlab.ListAgentsOptions{
		Page:    int(o.page),
		PerPage: int(o.perPage),
	})
	if err != nil {
		return err
	}

	title := utils.NewListTitle("agent")
	title.RepoName = repo.FullName()
	title.Page = int(o.page)
	title.CurrentPageTotal = len(agents)
	err = o.io.StartPager()
	if err != nil {
		return err
	}
	defer o.io.StopPager()

	o.io.LogInfof("%s\n%s\n", title.Describe(), agentutils.DisplayAllAgents(o.io, agents))
	return nil
}
