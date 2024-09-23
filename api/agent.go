package api

import (
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/xanzy/go-gitlab"
)

// agentTokenLimit specifies the maximal amount of agent tokens that can be active per agent at any given time.
const agentTokenLimit = 2

var AgentNotFoundErr = errors.New("agent not found")

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

var GetAgentByName = func(client *gitlab.Client, projectID interface{}, agentName string) (*gitlab.Agent, error) {
	opts := &gitlab.ListAgentsOptions{
		Page:    1,
		PerPage: 100,
	}

	for opts.Page != 0 {
		paginatedAgents, resp, err := client.ClusterAgents.ListAgents(projectID, opts)
		if err != nil {
			return nil, err
		}

		for _, agent := range paginatedAgents {
			if agent.Name == agentName {
				// found
				return agent, nil
			}
		}
		opts.Page = resp.NextPage
	}

	return nil, AgentNotFoundErr
}

var RegisterAgent = func(client *gitlab.Client, projectID interface{}, agentName string) (*gitlab.Agent, error) {
	if client == nil {
		client = apiClient.Lab()
	}

	agent, _, err := client.ClusterAgents.RegisterAgent(projectID, &gitlab.RegisterAgentOptions{Name: gitlab.Ptr(agentName)})
	if err != nil {
		return nil, err
	}

	return agent, nil
}

var CreateAgentToken = func(client *gitlab.Client, projectID interface{}, agentID int, recreateOnLimit bool) (*gitlab.AgentToken, bool /* recreated */, error) {
	recreated := false

	if recreateOnLimit {
		tokens, _, err := client.ClusterAgents.ListAgentTokens(projectID, agentID, &gitlab.ListAgentTokensOptions{PerPage: agentTokenLimit})
		if err != nil {
			return nil, false, err
		}
		if len(tokens) == agentTokenLimit {
			slices.SortFunc(tokens, agentTokenSortFunc)
			longestUnusedToken := tokens[0]

			_, err := client.ClusterAgents.RevokeAgentToken(projectID, agentID, longestUnusedToken.ID)
			if err != nil {
				return nil, false, err
			}
			recreated = true
		}
	}

	// create new token
	token, _, err := client.ClusterAgents.CreateAgentToken(projectID, agentID, &gitlab.CreateAgentTokenOptions{
		Name:        gitlab.Ptr(fmt.Sprintf("glab-bootstrap-%d", time.Now().UTC().Unix())),
		Description: gitlab.Ptr("Created by the `glab cluster agent bootstrap command"),
	})
	return token, recreated, err
}

func agentTokenSortFunc(a, b *gitlab.AgentToken) int {
	if a.LastUsedAt == nil {
		return 1
	}
	if b.LastUsedAt == nil {
		return -1
	}
	return a.LastUsedAt.Compare(*b.LastUsedAt)
}
