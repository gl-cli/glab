package bootstrap

import (
	"encoding/base64"
	"errors"
	"fmt"
	"slices"

	"github.com/xanzy/go-gitlab"
	glab_api "gitlab.com/gitlab-org/cli/api"
	"gopkg.in/yaml.v3"
)

var _ API = (*apiWrapper)(nil)

func NewAPI(client *gitlab.Client, projectID any) API {
	return &apiWrapper{client: client, projectID: projectID}
}

type apiWrapper struct {
	client    *gitlab.Client
	projectID any
}

func (a *apiWrapper) GetDefaultBranch() (string, error) {
	project, err := glab_api.GetProject(a.client, a.projectID)
	if err != nil {
		return "", err
	}
	return project.DefaultBranch, nil
}

func (a *apiWrapper) GetAgentByName(name string) (*gitlab.Agent, error) {
	return glab_api.GetAgentByName(a.client, a.projectID, name)
}

func (a *apiWrapper) RegisterAgent(name string) (*gitlab.Agent, error) {
	return glab_api.RegisterAgent(a.client, a.projectID, name)
}

type agentConfig struct {
	UserAccess *agentConfigUserAccess `yaml:"user_access"`
}

type agentConfigUserAccess struct {
	AccessAs *agentConfigAccessAs  `yaml:"access_as"`
	Projects []*agentConfigProject `yaml:"projects"`
}

type agentConfigAccessAs struct {
	Agent struct{}
}

type agentConfigProject struct {
	ID string `yaml:"id"`
}

func (a *apiWrapper) ConfigureAgent(agent *gitlab.Agent, branch string) error {
	configPath := fmt.Sprintf(".gitlab/agents/%s/config.yaml", agent.Name)
	file, err := glab_api.GetFile(a.client, a.projectID, configPath, branch)
	if err != nil && !glab_api.Is404(err) {
		return err
	}

	cfg := agentConfig{}
	if glab_api.Is404(err) {
		cfg.UserAccess = &agentConfigUserAccess{
			AccessAs: &agentConfigAccessAs{Agent: struct{}{}},
			Projects: []*agentConfigProject{
				{
					ID: agent.ConfigProject.PathWithNamespace,
				},
			},
		}

		configuredContent, err := yaml.Marshal(cfg)
		if err != nil {
			return err
		}

		return glab_api.CreateFile(a.client, a.projectID, configPath, configuredContent, branch)
	} else {
		content, err := base64.StdEncoding.DecodeString(file.Content)
		if err != nil {
			return err
		}

		err = yaml.Unmarshal(content, &cfg)
		if err != nil {
			return err
		}

		if !slices.ContainsFunc(cfg.UserAccess.Projects, func(p *agentConfigProject) bool { return p.ID == agent.ConfigProject.PathWithNamespace }) {
			cfg.UserAccess.Projects = append(cfg.UserAccess.Projects, &agentConfigProject{ID: agent.ConfigProject.PathWithNamespace})
		}

		configuredContent, err := yaml.Marshal(cfg)
		if err != nil {
			return err
		}

		return glab_api.UpdateFile(a.client, a.projectID, configPath, configuredContent, branch)
	}
}

func (a *apiWrapper) ConfigureEnvironment(agentID int, name string, kubernetesNamespace string, fluxResourcePath string) error {
	env, err := a.getEnvironmentByName(name)
	if err != nil {
		return err
	}

	if env == nil {
		_, _, err := a.client.Environments.CreateEnvironment(a.projectID, &gitlab.CreateEnvironmentOptions{
			Name:                gitlab.Ptr(name),
			ClusterAgentID:      gitlab.Ptr(agentID),
			KubernetesNamespace: gitlab.Ptr(kubernetesNamespace),
			FluxResourcePath:    gitlab.Ptr(fluxResourcePath),
		})
		return err
	} else {
		_, _, err := a.client.Environments.EditEnvironment(a.projectID, env.ID, &gitlab.EditEnvironmentOptions{
			Name:                gitlab.Ptr(name),
			ClusterAgentID:      gitlab.Ptr(agentID),
			KubernetesNamespace: gitlab.Ptr(kubernetesNamespace),
			FluxResourcePath:    gitlab.Ptr(fluxResourcePath),
		})
		return err
	}
}

func (a *apiWrapper) CreateAgentToken(agentID int) (*gitlab.AgentToken, error) {
	token, _, err := glab_api.CreateAgentToken(a.client, a.projectID, agentID, true)
	return token, err
}

func (a *apiWrapper) SyncFile(f file, branch string) error {
	return glab_api.SyncFile(a.client, a.projectID, f.path, f.content, branch)
}

func (a *apiWrapper) GetKASAddress() (string, error) {
	metadata, err := glab_api.GetMetadata(a.client)
	if err != nil {
		return "", err
	}

	if !metadata.KAS.Enabled {
		return "", errors.New("KAS is not configured in this GitLab instance. Please contact your administrator.")
	}

	return metadata.KAS.ExternalURL, nil
}

func (a *apiWrapper) getEnvironmentByName(name string) (*gitlab.Environment, error) {
	opts := &gitlab.ListEnvironmentsOptions{
		Name: gitlab.Ptr(name),
	}

	envs, _, err := a.client.Environments.ListEnvironments(a.projectID, opts)
	if err != nil {
		if glab_api.Is404(err) {
			return nil, nil
		}
		return nil, err
	}

	if len(envs) == 0 {
		return nil, nil
	}

	return envs[0], nil
}
