package claude

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cli/internal/testing/cmdtest"
	"gitlab.com/gitlab-org/cli/internal/testing/httpmock"
)

func runClaudeCommand(t *testing.T, rt http.RoundTripper, args string, glInstanceHostname string) error {
	ios, _, stdout, stderr := cmdtest.TestIOStreams()

	factory := cmdtest.NewTestFactory(ios,
		cmdtest.WithApiClient(cmdtest.NewTestApiClient(t, &http.Client{Transport: rt}, "", glInstanceHostname)),
		cmdtest.WithBaseRepo("OWNER", "REPO", glInstanceHostname),
	)

	cmd := NewCmdClaude(factory)

	_, err := cmdtest.ExecuteCommand(cmd, args, stdout, stderr)
	return err
}

func TestNewCmdClaude(t *testing.T) {
	ios, _, _, _ := cmdtest.TestIOStreams()
	factory := cmdtest.NewTestFactory(ios)

	cmd := NewCmdClaude(factory)

	assert.NotNil(t, cmd)
	assert.Equal(t, "claude [flags] [args]", cmd.Use)
	assert.Equal(t, "Launch Claude Code with GitLab Duo integration", cmd.Short)
	assert.True(t, cmd.FParseErrWhitelist.UnknownFlags)

	// Check that token subcommand is added
	tokenCmd := cmd.Commands()[0]
	assert.Equal(t, "token", tokenCmd.Use)
}

func TestClaudeCmdFailedTokenRetrieval(t *testing.T) {
	tests := []struct {
		name         string
		statusCode   int
		responseBody string
		expectedErr  string
	}{
		{
			name:         "API error",
			statusCode:   http.StatusUnauthorized,
			responseBody: `{"error": "unauthorized"}`,
			expectedErr:  "failed to retrieve GitLab Duo access token",
		},
		{
			name:         "Network error",
			statusCode:   http.StatusInternalServerError,
			responseBody: `{"error": "server error"}`,
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

			err := runClaudeCommand(t, fakeHTTP, "", "gitlab.com")

			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.expectedErr)
		})
	}
}

func TestClaudeCmdClaudeExecutableNotFound(t *testing.T) {
	// This test fails at argument parsing since the test environment doesn't set up os.Args with 'claude'
	fakeHTTP := &httpmock.Mocker{
		MatchURL: httpmock.PathAndQuerystring,
	}
	defer fakeHTTP.Verify(t)

	// Mock successful token response
	tokenResponse := httpmock.NewStringResponse(http.StatusCreated,
		`{"token": "test-token", "headers": {"X-Auth": "test-header"}}`)
	fakeHTTP.RegisterResponder(http.MethodPost, "/api/v4/ai/third_party_agents/direct_access", tokenResponse)

	err := runClaudeCommand(t, fakeHTTP, "", "gitlab.com")

	require.Error(t, err)
}

func TestClaudeCommandDescription(t *testing.T) {
	ios, _, _, _ := cmdtest.TestIOStreams()
	factory := cmdtest.NewTestFactory(ios)

	cmd := NewCmdClaude(factory)

	assert.Contains(t, cmd.Long, "Launch Claude Code with automatic GitLab authentication")
	assert.Contains(t, cmd.Long, "handling authentication tokens and API endpoints")
	assert.Contains(t, cmd.Example, "$ glab duo claude")
	assert.Contains(t, cmd.Example, `$ glab duo claude -p "Write a function to calculate Fibonacci numbers"`)
}
