package claude

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	gitlab "gitlab.com/gitlab-org/api/client-go"
)

const (
	// Standard claude environment variables defined [here](https://docs.anthropic.com/en/docs/claude-code/settings#environment-variables)
	EnvAnthropicCustomHeaders = "ANTHROPIC_CUSTOM_HEADERS"
	EnvAnthropicBaseURL       = "ANTHROPIC_BASE_URL"
	EnvAnthropicModel         = "ANTHROPIC_MODEL"
	EnvAnthropicAuthToken     = "ANTHROPIC_AUTH_TOKEN"

	// Default model: This needs to be configured as we don't support
	// all models
	DefaultClaudeModel = "claude-sonnet-4-20250514"

	// Claude executable name
	ClaudeExecutable = "claude"

	// Settings configuration
	ClaudeSettingsDir = ".claude"
	SettingsFileName  = "settings.json"
	APIKeyHelperKey   = "apiKeyHelper"

	CloudConnectorUrl = "https://cloud.gitlab.com/ai/v1/proxy/anthropic"
)

// getHeaderEnv formats headers as environment variable value.
func getHeaderEnv(headers map[string]string) string {
	var headerParts []string
	for k, v := range headers {
		headerParts = append(headerParts, fmt.Sprintf("%s: %s", k, v))
	}
	return strings.Join(headerParts, "\n")
}

// fetchDirectAccessToken retrieves a direct access token from GitLab AI service.
func fetchDirectAccessToken(client *gitlab.Client) (*DirectAccessResponse, error) {
	req, err := client.NewRequest(http.MethodPost, "ai/third_party_agents/direct_access", nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for direct access token: %w", err)
	}

	var response DirectAccessResponse
	resp, err := client.Do(req, &response)
	if err != nil {
		if gitlab.HasStatusCode(err, http.StatusForbidden) {
			return nil, fmt.Errorf("failed to execute direct access token request: %w (your user most likely isn't enabled for the `agent_platform_claude_code` feature flag, please contact your GitLab administrator to enable it)", err)
		} else {
			return nil, fmt.Errorf("failed to execute direct access token request: %w", err)
		}
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

// setClaudeSettings configures Claude settings to use this binary as the API key helper.
// Returns true if successful, false otherwise.
func setClaudeSettings() bool {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return false
	}

	settingsPath := filepath.Join(homeDir, ClaudeSettingsDir, SettingsFileName)

	// Ensure the settings directory exists
	if err := ensureSettingsDir(settingsPath); err != nil {
		return false
	}

	// Read existing settings
	settings, err := readSettings(settingsPath)
	if err != nil {
		return false
	}

	// Get current binary path
	exePath, err := os.Executable()
	if err != nil {
		return false
	}

	// Update apiKeyHelper setting
	settings[APIKeyHelperKey] = fmt.Sprintf("%s duo claude token", exePath)

	// Write updated settings
	return writeSettings(settingsPath, settings)
}

// ensureSettingsDir creates the settings directory and file if they don't exist.
func ensureSettingsDir(settingsPath string) error {
	if _, err := os.Stat(settingsPath); os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(settingsPath), 0o755); err != nil {
			return fmt.Errorf("failed to create settings directory: %w", err)
		}
		file, err := os.Create(settingsPath)
		if err != nil {
			return fmt.Errorf("failed to create settings file: %w", err)
		}
		file.Close()
	}
	return nil
}

// readSettings reads and parses the Claude settings file.
func readSettings(settingsPath string) (map[string]any, error) {
	settingsFile, err := os.ReadFile(settingsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read settings file: %w", err)
	}

	var settings map[string]any
	if len(settingsFile) > 0 {
		if err := json.Unmarshal(settingsFile, &settings); err != nil {
			return nil, fmt.Errorf("failed to parse settings JSON: %w", err)
		}
	} else {
		settings = make(map[string]any)
	}

	return settings, nil
}

// writeSettings writes the settings to the Claude settings file.
func writeSettings(settingsPath string, settings map[string]any) bool {
	updatedSettings, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return false
	}

	if err := os.WriteFile(settingsPath, updatedSettings, 0o644); err != nil {
		return false
	}

	return true
}

// validateClaudeExecutable checks if the Claude executable exists and is accessible.
func validateClaudeExecutable() error {
	_, err := exec.LookPath(ClaudeExecutable)
	if err != nil {
		return fmt.Errorf("claude executable not found in PATH: %w", err)
	}
	return nil
}

// extractClaudeArgs extracts arguments after "claude" from os.Args.
func extractClaudeArgs() ([]string, error) {
	osArgs := os.Args

	// Find the index where "claude" appears in the arguments
	claudeIndex := -1
	for i, arg := range osArgs {
		if arg == "claude" {
			claudeIndex = i
			break
		}
	}

	if claudeIndex == -1 {
		return nil, fmt.Errorf("could not find 'claude' in command arguments")
	}

	// Return all arguments after "claude"
	return osArgs[claudeIndex+1:], nil
}
