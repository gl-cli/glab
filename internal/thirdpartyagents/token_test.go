package thirdpartyagents

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/cli/internal/testing/httpmock"
)

func TestFetchDirectAccessToken(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		responseBody   string
		expectedError  string
		expectedToken  string
		expectedHeader string
	}{
		{
			name:           "successful request",
			statusCode:     http.StatusCreated,
			responseBody:   `{"token": "test-token", "headers": {"X-Auth": "value"}}`,
			expectedToken:  "test-token",
			expectedHeader: "value",
		},
		{
			name:          "wrong status code",
			statusCode:    http.StatusBadRequest,
			responseBody:  `{"error": "bad request"}`,
			expectedError: "failed to execute direct access token request",
		},
		{
			name:          "invalid JSON",
			statusCode:    http.StatusCreated,
			responseBody:  `invalid json`,
			expectedError: "failed to execute direct access token request",
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

			client, err := gitlab.NewClient("", gitlab.WithHTTPClient(&http.Client{Transport: fakeHTTP}))
			require.NoError(t, err)

			result, err := FetchDirectAccessToken(client)

			if tc.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tc.expectedToken, result.Token)
				if tc.expectedHeader != "" {
					assert.Equal(t, tc.expectedHeader, result.Headers["X-Auth"])
				}
			}
		})
	}
}
