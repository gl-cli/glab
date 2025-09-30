package claude

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

// getHomeDir returns the home directory, preferring the HOME environment variable
// over os.UserHomeDir() to support test environments.
func getHomeDir() (string, error) {
	// First try HOME environment variable (works in test environments)
	if homeDir := os.Getenv("HOME"); homeDir != "" {
		return homeDir, nil
	}
	// Fall back to os.UserHomeDir()
	return os.UserHomeDir()
}

// setClaudeSettings configures Claude settings to use this binary as the API key helper.
// Returns true if successful, false otherwise.
func setClaudeSettings() bool {
	homeDir, err := getHomeDir()
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
