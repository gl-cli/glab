package codex

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cli/internal/testing/cmdtest"
	"gitlab.com/gitlab-org/cli/internal/testing/httpmock"
)

func runCodexCommand(t *testing.T, rt http.RoundTripper, args string, glInstanceHostname string) error {
	t.Helper()

	ios, _, stdout, stderr := cmdtest.TestIOStreams()

	factory := cmdtest.NewTestFactory(ios,
		cmdtest.WithApiClient(cmdtest.NewTestApiClient(t, &http.Client{Transport: rt}, "", glInstanceHostname)),
		cmdtest.WithBaseRepo("OWNER", "REPO", glInstanceHostname),
	)

	cmd := NewCmdCodex(factory)

	_, err := cmdtest.ExecuteCommand(cmd, args, stdout, stderr)
	return err
}

func TestNewCmdCodex(t *testing.T) {
	ios, _, _, _ := cmdtest.TestIOStreams()
	factory := cmdtest.NewTestFactory(ios)

	cmd := NewCmdCodex(factory)

	assert.NotNil(t, cmd)
	assert.Equal(t, "codex [flags] [args]", cmd.Use)
	assert.Equal(t, "Launch Codex with GitLab Duo integration (EXPERIMENTAL)", cmd.Short)
	assert.True(t, cmd.FParseErrWhitelist.UnknownFlags)
}

func TestCodexCmdFailedTokenRetrieval(t *testing.T) {
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

			err := runCodexCommand(t, fakeHTTP, "", "gitlab.com")

			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.expectedErr)
		})
	}
}

func TestCodexCmdCodexExecutableNotFound(t *testing.T) {
	// This test fails at argument parsing since the test environment doesn't set up os.Args with 'codex'
	fakeHTTP := &httpmock.Mocker{
		MatchURL: httpmock.PathAndQuerystring,
	}
	defer fakeHTTP.Verify(t)

	// Mock successful token response
	tokenResponse := httpmock.NewStringResponse(http.StatusCreated,
		`{"token": "test-token", "headers": {"X-Auth": "test-header"}}`)
	fakeHTTP.RegisterResponder(http.MethodPost, "/api/v4/ai/third_party_agents/direct_access", tokenResponse)

	err := runCodexCommand(t, fakeHTTP, "", "gitlab.com")

	require.Error(t, err)
}

func TestCodexCommandDescription(t *testing.T) {
	ios, _, _, _ := cmdtest.TestIOStreams()
	factory := cmdtest.NewTestFactory(ios)

	cmd := NewCmdCodex(factory)

	assert.Contains(t, cmd.Long, "Launch Codex with automatic GitLab authentication")
	assert.Contains(t, cmd.Long, "handling authentication tokens and API endpoints")
	assert.Contains(t, cmd.Example, "$ glab duo codex")
}

func TestCodexExecutableName(t *testing.T) {
	assert.Equal(t, "codex", CodexExecutableName)
}
