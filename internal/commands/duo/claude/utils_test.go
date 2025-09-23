package claude

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/cli/internal/testing/httpmock"
)

// isolateHomeDir creates a temporary directory and sets it as HOME for the test,
// automatically restoring the original HOME when the test completes.
// Returns the path to the temporary directory.
func isolateHomeDir(t *testing.T) string {
	tempDir := t.TempDir()
	t.Setenv("HOME", tempDir)
	return tempDir
}

func TestGetHeaderEnv(t *testing.T) {
	tests := []struct {
		name     string
		headers  map[string]string
		expected string
	}{
		{
			name:     "empty headers",
			headers:  map[string]string{},
			expected: "",
		},
		{
			name: "single header",
			headers: map[string]string{
				"Authorization": "Bearer token123",
			},
			expected: "Authorization: Bearer token123",
		},
		{
			name: "multiple headers",
			headers: map[string]string{
				"Authorization": "Bearer token123",
				"X-Custom":      "value",
			},
			// Note: map iteration order is not guaranteed, so we check both orders
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := getHeaderEnv(tc.headers)
			if len(tc.headers) == 0 {
				assert.Equal(t, tc.expected, result)
			} else if len(tc.headers) == 1 {
				assert.Equal(t, tc.expected, result)
			} else {
				// For multiple headers, just check that all expected parts are present
				for k, v := range tc.headers {
					assert.Contains(t, result, k+": "+v)
				}
			}
		})
	}
}

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

			result, err := fetchDirectAccessToken(client)

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

func TestValidateClaudeExecutable(t *testing.T) {
	// Test when executable doesn't exist
	originalPath := os.Getenv("PATH")
	defer t.Setenv("PATH", originalPath)

	// Set PATH to empty to ensure claude is not found
	t.Setenv("PATH", "")

	err := validateClaudeExecutable()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "claude executable not found in PATH")
}

func TestExtractClaudeArgs(t *testing.T) {
	tests := []struct {
		name          string
		osArgs        []string
		expectedArgs  []string
		expectedError string
	}{
		{
			name:         "claude with args",
			osArgs:       []string{"glab", "duo", "claude", "--help", "some", "args"},
			expectedArgs: []string{"--help", "some", "args"},
		},
		{
			name:         "claude without args",
			osArgs:       []string{"glab", "duo", "claude"},
			expectedArgs: []string{},
		},
		{
			name:          "no claude in args",
			osArgs:        []string{"glab", "duo", "ask", "something"},
			expectedError: "could not find 'claude' in command arguments",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Save original os.Args
			originalArgs := os.Args
			defer func() { os.Args = originalArgs }()

			// Set test args
			os.Args = tc.osArgs

			result, err := extractClaudeArgs(nil)

			if tc.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedArgs, result)
			}
		})
	}
}

func TestSetClaudeSettings(t *testing.T) {
	// Isolate HOME directory for testing
	tempDir := isolateHomeDir(t)

	result := setClaudeSettings()
	assert.True(t, result)

	// Verify the settings file was created
	settingsPath := filepath.Join(tempDir, ClaudeSettingsDir, SettingsFileName)
	assert.FileExists(t, settingsPath)

	// Verify the content
	content, err := os.ReadFile(settingsPath)
	require.NoError(t, err)

	var settings map[string]any
	err = json.Unmarshal(content, &settings)
	require.NoError(t, err)

	apiKeyHelper, exists := settings[APIKeyHelperKey]
	assert.True(t, exists)
	assert.Contains(t, apiKeyHelper.(string), "duo claude token")
}

func TestEnsureSettingsDir(t *testing.T) {
	tempDir := t.TempDir()
	settingsPath := filepath.Join(tempDir, "test", SettingsFileName)

	err := ensureSettingsDir(settingsPath)
	assert.NoError(t, err)

	// Verify directory was created
	assert.DirExists(t, filepath.Dir(settingsPath))
	assert.FileExists(t, settingsPath)
}

func TestReadSettings(t *testing.T) {
	tests := []struct {
		name          string
		fileContent   string
		expectedError string
		expectedData  map[string]any
	}{
		{
			name:         "valid JSON",
			fileContent:  `{"apiKeyHelper": "test-value"}`,
			expectedData: map[string]any{"apiKeyHelper": "test-value"},
		},
		{
			name:         "empty file",
			fileContent:  "",
			expectedData: map[string]any{},
		},
		{
			name:          "invalid JSON",
			fileContent:   `{"invalid": json}`,
			expectedError: "failed to parse settings JSON",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tempDir := t.TempDir()
			settingsPath := filepath.Join(tempDir, "settings.json")

			err := os.WriteFile(settingsPath, []byte(tc.fileContent), 0o644)
			require.NoError(t, err)

			result, err := readSettings(settingsPath)

			if tc.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedData, result)
			}
		})
	}
}

func TestWriteSettings(t *testing.T) {
	tempDir := t.TempDir()
	settingsPath := filepath.Join(tempDir, "settings.json")

	settings := map[string]any{
		"apiKeyHelper": "test-value",
		"otherKey":     "other-value",
	}

	result := writeSettings(settingsPath, settings)
	assert.True(t, result)

	// Verify the file was written correctly
	content, err := os.ReadFile(settingsPath)
	require.NoError(t, err)

	var readSettings map[string]any
	err = json.Unmarshal(content, &readSettings)
	require.NoError(t, err)

	assert.Equal(t, settings, readSettings)
}

func TestWriteSettingsInvalidPath(t *testing.T) {
	// Try to write to an invalid path (directory doesn't exist and can't be created)
	invalidPath := "/root/nonexistent/settings.json"

	settings := map[string]any{"test": "value"}

	result := writeSettings(invalidPath, settings)
	assert.False(t, result)
}
