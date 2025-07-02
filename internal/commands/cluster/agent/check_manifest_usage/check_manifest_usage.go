package check_manifest_usage

import (
	"fmt"

	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/cli/internal/api"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/internal/iostreams"
	"gitlab.com/gitlab-org/cli/internal/text"
	"gopkg.in/yaml.v3"
)

type options struct {
	group          string
	recursive      bool
	projectPerPage int
	projectPage    int
	agentPerPage   int
	agentPage      int

	apiClient func(repoHost string, cfg config.Config) (*api.Client, error)
	config    config.Config
	io        *iostreams.IOStreams
}

type AgentConfig struct {
	GitOps struct {
		ManifestProjects []struct{} `yaml:"manifest_projects,omitempty"`
	} `yaml:"gitops,omitempty"`
}

func NewCmdCheckManifestUsage(f cmdutils.Factory) *cobra.Command {
	opts := &options{
		io:        f.IO(),
		apiClient: f.ApiClient,
		config:    f.Config(),
	}
	checkManifestUsageCmd := &cobra.Command{
		Use:   "check_manifest_usage [flags]",
		Short: `Check agent configuration files for built-in GitOps manifests usage. (EXPERIMENTAL.)`,
		Long: `Checks the descendants of a group for registered agents with configuration files that rely on the deprecated GitOps manifests settings.
The output can be piped to a tab-separated value (TSV) file.
` + text.ExperimentalString,
		RunE: func(cmd *cobra.Command, args []string) error {
			return opts.run()
		},
	}
	// Boolean to authorize experimental features
	checkManifestUsageCmd.Flags().StringVarP(&opts.group, "group", "g", "", "Group ID to check.")
	cobra.CheckErr(checkManifestUsageCmd.MarkFlagRequired("group"))
	checkManifestUsageCmd.Flags().IntVarP(&opts.projectPage, "page", "p", 1, "Page number for projects.")
	checkManifestUsageCmd.Flags().IntVarP(&opts.projectPerPage, "per-page", "P", 30, "Number of projects to list per page.")
	checkManifestUsageCmd.Flags().IntVarP(&opts.agentPage, "agent-page", "a", 1, "Page number for projects.")
	checkManifestUsageCmd.Flags().IntVarP(&opts.agentPerPage, "agent-per-page", "A", 30, "Number of projects to list per page.")
	checkManifestUsageCmd.Flags().BoolVarP(&opts.recursive, "recursive", "r", false, "Recursively check subgroups.")

	return checkManifestUsageCmd
}

func (o *options) run() error {
	c, err := o.apiClient("", o.config)
	if err != nil {
		return err
	}
	client := c.Lab()

	// new line
	err = checkGroup(client, o.group, o)
	if err != nil {
		return err
	}

	if o.recursive {
		var groups []*gitlab.Group
		groups, _, err = listAllGroupsForGroup(client, o.group)
		if err != nil {
			return err
		}

		for _, group := range groups {
			err = checkGroup(client, group.FullPath, o)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func checkGroup(apiClient *gitlab.Client, group string, opts *options) error {
	var projects []*gitlab.Project
	var resp *gitlab.Response
	projects, resp, err := listAllProjectsForGroup(apiClient, group, *opts)
	if err != nil {
		return err
	}

	color := opts.io.Color()
	opts.io.Log(color.ProgressIcon(), fmt.Sprintf("Checking %d of %d projects (Page %d of %d)\n", len(projects), resp.TotalItems, resp.CurrentPage, resp.TotalPages))
	for _, prj := range projects {
		err = checkManifestUsageInProject(apiClient, opts, prj)
		if err != nil {
			return err
		}
	}

	opts.io.Log()
	return nil
}

func listAllGroupsForGroup(apiClient *gitlab.Client, group string) ([]*gitlab.Group, *gitlab.Response, error) {
	l := &gitlab.ListDescendantGroupsOptions{}
	return apiClient.Groups.ListDescendantGroups(group, l)
}

func listAllProjectsForGroup(apiClient *gitlab.Client, group string, opts options) ([]*gitlab.Project, *gitlab.Response, error) {
	l := &gitlab.ListGroupProjectsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: opts.projectPerPage,
			Page:    opts.projectPage,
		},
	}

	return apiClient.Groups.ListGroupProjects(group, l)
}

func checkManifestUsageInProject(apiClient *gitlab.Client, opts *options, project *gitlab.Project) error {
	color := opts.io.Color()
	opts.io.StartSpinner(fmt.Sprintf("Checking project %s for agents.\n", project.PathWithNamespace))
	defer opts.io.StopSpinner("")

	agents, _, err := apiClient.ClusterAgents.ListAgents(project.ID, &gitlab.ListAgentsOptions{
		Page:    opts.agentPage,
		PerPage: opts.agentPerPage,
	})
	if err != nil {
		return err
	}

	opts.io.Log(color.ProgressIcon(), fmt.Sprintf("Found %d agents.\n", len(agents)))
	for _, agent := range agents {
		found, err := agentUsesManifestProjects(apiClient, opts, agent)
		if err != nil {
			opts.io.Log(color.RedCheck(), "An error happened.", err)
			continue
		}
		if found {
			opts.io.LogInfo(fmt.Sprintf("%s\t%s\t%d", agent.ConfigProject.PathWithNamespace, agent.Name, 1))
		} else {
			opts.io.LogInfo(fmt.Sprintf("%s\t%s\t%d", agent.ConfigProject.PathWithNamespace, agent.Name, 0))
		}
	}

	return nil
}

func agentUsesManifestProjects(apiClient *gitlab.Client, opts *options, agent *gitlab.Agent) (bool, error) {
	color := opts.io.Color()
	opts.io.StartSpinner(fmt.Sprintf("Checking manifests of agent %s.\n", agent.Name))
	defer opts.io.StopSpinner("")

	// GetRawFile
	file, _, err := apiClient.RepositoryFiles.GetRawFile(agent.ConfigProject.ID, ".gitlab/agents/"+agent.Name+"/config.yaml", &gitlab.GetRawFileOptions{})
	if err != nil {
		opts.io.Log(color.WarnIcon(), fmt.Sprintf("Agent %s uses the default configuration.", agent.Name))
		return false, nil
	}

	// Check that gitops.manifest_projects json path does not exist in configFile
	var configData AgentConfig
	err = yaml.Unmarshal(file, &configData)
	if err != nil {
		opts.io.Log("Unmarshal error", fmt.Sprintf("%s\n", string(file)))
		return false, err
	}

	if len(configData.GitOps.ManifestProjects) == 0 {
		opts.io.Log(color.GreenCheck(), fmt.Sprintf("Agent %s does not have manifest projects configured.", agent.Name))
		return false, nil
	} else {
		opts.io.Log(color.FailedIcon(), fmt.Sprintf("Agent %s has %d manifest projects configured.", agent.Name, len(configData.GitOps.ManifestProjects)))
		return true, nil
	}
}
