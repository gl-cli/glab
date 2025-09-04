package claude

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cli/internal/api"
	"gitlab.com/gitlab-org/cli/internal/testing/cmdtest"
	"gitlab.com/gitlab-org/cli/internal/testing/httpmock"
	"gitlab.com/gitlab-org/cli/test"
)

func runTokenCommand(t *testing.T, rt http.RoundTripper, args string, glInstanceHostname string) (*test.CmdOut, *cmdtest.Factory, error) {
	ios, _, stdout, stderr := cmdtest.TestIOStreams()

	factory := cmdtest.NewTestFactory(ios,
		cmdtest.WithApiClient(cmdtest.NewTestApiClient(t, &http.Client{Transport: rt}, "", glInstanceHostname)),
		cmdtest.WithBaseRepo("OWNER", "REPO", glInstanceHostname),
	)

	cmd := NewCmdToken(factory)

	cmdOut, err := cmdtest.ExecuteCommand(cmd, args, stdout, stderr)

	return cmdOut, factory, err
}

func TestNewCmdToken(t *testing.T) {
	ios, _, _, _ := cmdtest.TestIOStreams()
	factory := cmdtest.NewTestFactory(ios)

	cmd := NewCmdToken(factory)

	assert.NotNil(t, cmd)
	assert.Equal(t, "token", cmd.Use)
	assert.Equal(t, "Generate GitLab Duo access token for Claude Code", cmd.Short)
	assert.True(t, cmd.FParseErrWhitelist.UnknownFlags)
	assert.Contains(t, cmd.Long, "Generate and display a GitLab Duo access token")
	assert.Contains(t, cmd.Long, "This token allows Claude Code to authenticate")
	assert.Contains(t, cmd.Example, "$ glab duo claude token")
}

func TestTokenCmdSuccessful(t *testing.T) {
	fakeHTTP := &httpmock.Mocker{
		MatchURL: httpmock.PathAndQuerystring,
	}
	defer fakeHTTP.Verify(t)

	tokenResponse := httpmock.NewStringResponse(http.StatusCreated,
		`{"token": "test-token-123", "headers": {"X-Auth": "test-header"}}`)
	fakeHTTP.RegisterResponder(http.MethodPost, "/api/v4/ai/third_party_agents/direct_access", tokenResponse)

	output, _, err := runTokenCommand(t, fakeHTTP, "", "gitlab.com")

	require.NoError(t, err)
	assert.Equal(t, "test-token-123\n", output.String())
	assert.Empty(t, output.Stderr())
}

func TestTokenCmdFailedTokenRetrieval(t *testing.T) {
	tests := []struct {
		name         string
		statusCode   int
		responseBody string
		expectedErr  string
	}{
		{
			name:         "unauthorized",
			statusCode:   http.StatusUnauthorized,
			responseBody: `{"error": "unauthorized"}`,
			expectedErr:  "failed to retrieve GitLab Duo access token",
		},
		{
			name:         "forbidden",
			statusCode:   http.StatusForbidden,
			responseBody: `{"error": "forbidden"}`,
			expectedErr:  "failed to retrieve GitLab Duo access token",
		},
		{
			name:         "server error",
			statusCode:   http.StatusInternalServerError,
			responseBody: `{"error": "internal server error"}`,
			expectedErr:  "failed to retrieve GitLab Duo access token",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fakeHTTP := &httpmock.Mocker{
				MatchURL: httpmock.PathAndQuerystring,
			}
			defer fakeHTTP.Verify(t)

			response := httpmock.NewStringResponse(tc.statusCode, tc.responseBody)
			fakeHTTP.RegisterResponder(http.MethodPost, "/api/v4/ai/third_party_agents/direct_access", response)

			_, _, err := runTokenCommand(t, fakeHTTP, "", "gitlab.com")

			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.expectedErr)
		})
	}
}

func TestTokenCmdWithDifferentHosts(t *testing.T) {
	tests := []struct {
		name     string
		hostname string
	}{
		{
			name:     "gitlab.com",
			hostname: "gitlab.com",
		},
		{
			name:     "custom host",
			hostname: "gitlab.example.com",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fakeHTTP := &httpmock.Mocker{
				MatchURL: httpmock.PathAndQuerystring,
			}
			defer fakeHTTP.Verify(t)

			tokenResponse := httpmock.NewStringResponse(http.StatusCreated,
				`{"token": "test-token", "headers": {"X-Auth": "test"}}`)
			fakeHTTP.RegisterResponder(http.MethodPost, "/api/v4/ai/third_party_agents/direct_access", tokenResponse)

			output, factory, err := runTokenCommand(t, fakeHTTP, "", tc.hostname)

			require.NoError(t, err)
			assert.Equal(t, "test-token\n", output.String())

			baseRepo, _ := factory.BaseRepo()
			assert.Equal(t, tc.hostname, baseRepo.RepoHost())
		})
	}
}

func TestTokenCmdAPIClientError(t *testing.T) {
	ios, _, _, _ := cmdtest.TestIOStreams()

	factory := cmdtest.NewTestFactory(ios,
		cmdtest.WithApiClient(nil), // This will cause an error
	)

	cmd := NewCmdToken(factory)

	// Override the apiClient to return an error
	factory.ApiClientStub = func(repoHost string) (*api.Client, error) {
		return nil, fmt.Errorf("api client creation failed")
	}

	_, err := cmd.ExecuteC()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "api client creation failed")
}
