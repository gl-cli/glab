package check_manifest_usage

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/xanzy/go-gitlab"
	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/pkg/iostreams"
	"gitlab.com/gitlab-org/cli/pkg/text"
	"gopkg.in/yaml.v3"
)

type Options struct {
	Group          string
	Recursive      bool
	ProjectPerPage int
	ProjectPage    int
	AgentPerPage   int
	AgentPage      int

	HTTPClient func() (*gitlab.Client, error)
	IO         *iostreams.IOStreams
}

type AgentConfig struct {
	GitOps struct {
		ManifestProjects []struct{} `yaml:"manifest_projects,omitempty"`
	} `yaml:"gitops,omitempty"`
}

func NewCmdCheckManifestUsage(f *cmdutils.Factory) *cobra.Command {
	opts := &Options{
		IO: f.IO,
	}
	checkManifestUsageCmd := &cobra.Command{
		Use:   "check_manifest_usage [flags]",
		Short: `Check agent configuration files for built-in GitOps manifests usage. (EXPERIMENTAL.)`,
		Long: `Checks the descendants of a group for registered agents with configuration files that rely on the deprecated GitOps manifests settings.
The output can be piped to a tab-separated value (TSV) file.
` + text.ExperimentalString,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.HTTPClient = f.HttpClient

			return checkManifestUsageInGroup(opts)
		},
	}
	// Boolean to authorize experimental features
	checkManifestUsageCmd.Flags().StringVarP(&opts.Group, "group", "g", "", "Group ID to check.")
	cobra.CheckErr(checkManifestUsageCmd.MarkFlagRequired("group"))
	checkManifestUsageCmd.Flags().IntVarP(&opts.ProjectPage, "page", "p", 1, "Page number for projects.")
	checkManifestUsageCmd.Flags().IntVarP(&opts.ProjectPerPage, "per-page", "P", 30, "Number of projects to list per page.")
	checkManifestUsageCmd.Flags().IntVarP(&opts.AgentPage, "agent-page", "a", 1, "Page number for projects.")
	checkManifestUsageCmd.Flags().IntVarP(&opts.AgentPerPage, "agent-per-page", "A", 30, "Number of projects to list per page.")
	checkManifestUsageCmd.Flags().BoolVarP(&opts.Recursive, "recursive", "r", false, "Recursively check subgroups.")

	return checkManifestUsageCmd
}

func checkManifestUsageInGroup(opts *Options) error {
	apiClient, err := opts.HTTPClient()
	if err != nil {
		return err
	}

	// new line
	err = checkGroup(apiClient, opts.Group, opts)
	if err != nil {
		return err
	}

	if opts.Recursive {
		var groups []*gitlab.Group
		groups, _, err = listAllGroupsForGroup(apiClient, opts.Group)
		if err != nil {
			return err
		}

		for _, group := range groups {
			err = checkGroup(apiClient, group.FullPath, opts)
			if err != nil {
				return err
			}
		}

	}

	return nil
}

func checkGroup(apiClient *gitlab.Client, group string, opts *Options) error {
	var projects []*gitlab.Project
	var resp *gitlab.Response
	projects, resp, err := listAllProjectsForGroup(apiClient, group, *opts)
	if err != nil {
		return err
	}

	color := opts.IO.Color()
	opts.IO.Log(color.ProgressIcon(), fmt.Sprintf("Checking %d of %d projects (Page %d of %d)\n", len(projects), resp.TotalItems, resp.CurrentPage, resp.TotalPages))
	for _, prj := range projects {
		err = checkManifestUsageInProject(apiClient, opts, prj)
		if err != nil {
			return err
		}
	}

	opts.IO.Log()
	return nil
}

func listAllGroupsForGroup(apiClient *gitlab.Client, group string) ([]*gitlab.Group, *gitlab.Response, error) {
	l := &gitlab.ListDescendantGroupsOptions{}
	return apiClient.Groups.ListDescendantGroups(group, l)
}

func listAllProjectsForGroup(apiClient *gitlab.Client, group string, opts Options) ([]*gitlab.Project, *gitlab.Response, error) {
	l := &gitlab.ListGroupProjectsOptions{}

	l.PerPage = opts.ProjectPerPage
	l.Page = opts.ProjectPage

	return apiClient.Groups.ListGroupProjects(group, l)
}

func checkManifestUsageInProject(apiClient *gitlab.Client, opts *Options, project *gitlab.Project) error {
	color := opts.IO.Color()
	opts.IO.StartSpinner(fmt.Sprintf("Checking project %s for agents.\n", project.PathWithNamespace))
	defer opts.IO.StopSpinner("")

	agents, err := api.ListAgents(apiClient, project.ID, &gitlab.ListAgentsOptions{
		Page:    opts.AgentPage,
		PerPage: opts.AgentPerPage,
	})
	if err != nil {
		return err
	}

	opts.IO.Log(color.ProgressIcon(), fmt.Sprintf("Found %d agents.\n", len(agents)))
	for _, agent := range agents {
		found, err := agentUsesManifestProjects(apiClient, opts, agent)
		if err != nil {
			opts.IO.Log(color.RedCheck(), "An error happened.", err)
			continue
		}
		if found {
			opts.IO.LogInfo(fmt.Sprintf("%s\t%s\t%d", agent.ConfigProject.PathWithNamespace, agent.Name, 1))
		} else {
			opts.IO.LogInfo(fmt.Sprintf("%s\t%s\t%d", agent.ConfigProject.PathWithNamespace, agent.Name, 0))
		}
	}

	return nil
}

func agentUsesManifestProjects(apiClient *gitlab.Client, opts *Options, agent *gitlab.Agent) (bool, error) {
	color := opts.IO.Color()
	opts.IO.StartSpinner(fmt.Sprintf("Checking manifests of agent %s.\n", agent.Name))
	defer opts.IO.StopSpinner("")

	// GetRawFile
	file, _, err := apiClient.RepositoryFiles.GetRawFile(agent.ConfigProject.ID, ".gitlab/agents/"+agent.Name+"/config.yaml", &gitlab.GetRawFileOptions{})
	if err != nil {
		opts.IO.Log(color.WarnIcon(), fmt.Sprintf("Agent %s uses the default configuration.", agent.Name))
		return false, nil
	}

	// Check that gitops.manifest_projects json path does not exist in configFile
	var configData AgentConfig
	err = yaml.Unmarshal(file, &configData)
	if err != nil {
		opts.IO.Log("Unmarshal error", fmt.Sprintf("%s\n", string(file)))
		return false, err
	}

	if len(configData.GitOps.ManifestProjects) == 0 {
		opts.IO.Log(color.GreenCheck(), fmt.Sprintf("Agent %s does not have manifest projects configured.", agent.Name))
		return false, nil
	} else {
		opts.IO.Log(color.FailedIcon(), fmt.Sprintf("Agent %s has %d manifest projects configured.", agent.Name, len(configData.GitOps.ManifestProjects)))
		return true, nil
	}
}
