package api

import "github.com/xanzy/go-gitlab"

var ListAgents = func(client *gitlab.Client, projectID interface{}, opts *gitlab.ListAgentsOptions) ([]*gitlab.Agent, error) {
	if client == nil {
		client = apiClient.Lab()
	}

	agents, _, err := client.ClusterAgents.ListAgents(projectID, opts)
	if err != nil {
		return nil, err
	}

	return agents, nil
}

var GetAgent = func(client *gitlab.Client, projectID interface{}, agentID int) (*gitlab.Agent, error) {
	if client == nil {
		client = apiClient.Lab()
	}

	agent, _, err := client.ClusterAgents.GetAgent(projectID, agentID)
	if err != nil {
		return nil, err
	}

	return agent, nil
}
