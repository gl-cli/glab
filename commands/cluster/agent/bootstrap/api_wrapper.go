package bootstrap

import (
	"github.com/xanzy/go-gitlab"
	glab_api "gitlab.com/gitlab-org/cli/api"
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

func (a *apiWrapper) CreateAgentToken(agentID int) (*gitlab.AgentToken, error) {
	token, _, err := glab_api.CreateAgentToken(a.client, a.projectID, agentID, true)
	return token, err
}

func (a *apiWrapper) SyncFile(f file, branch string) error {
	return glab_api.SyncFile(a.client, a.projectID, f.path, f.content, branch)
}
