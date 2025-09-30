package thirdpartyagents

import (
	"fmt"
	"net/http"

	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// fetchDirectAccessToken retrieves a direct access token from GitLab AI service.
func FetchDirectAccessToken(client *gitlab.Client) (*DirectAccessResponse, error) {
	req, err := client.NewRequest(http.MethodPost, "ai/third_party_agents/direct_access", nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for direct access token: %w", err)
	}

	var response DirectAccessResponse
	resp, err := client.Do(req, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to execute direct access token request: %w", err)
	}

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("failed to retrieve direct access token: received status code %d instead of %d", resp.StatusCode, http.StatusCreated)
	}

	return &response, nil
}

// DirectAccessResponse represents the response from GitLab direct access token API.
type DirectAccessResponse struct {
	Headers map[string]string `json:"headers"`
	Token   string            `json:"token"`
}
