package codex

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateCodexConfigArgs(t *testing.T) {
	tests := []struct {
		name     string
		headers  map[string]string
		expected []string
	}{
		{
			name:    "empty headers",
			headers: map[string]string{},
			expected: []string{
				"--config", `model_provider="gitlab"`,
				"--config", `model_providers.gitlab.name="GitLab Managed Codex"`,
				"--config", `model_providers.gitlab.base_url="` + CloudConnectorUrl + `"`,
				"--config", `model_providers.gitlab.env_key="OPENAI_API_KEY"`,
				"--config", `model_providers.gitlab.wire_api="responses"`,
			},
		},
		{
			name: "single header",
			headers: map[string]string{
				"Authorization": "Bearer token123",
			},
			expected: []string{
				"--config", `model_provider="gitlab"`,
				"--config", `model_providers.gitlab.name="GitLab Managed Codex"`,
				"--config", `model_providers.gitlab.base_url="` + CloudConnectorUrl + `"`,
				"--config", `model_providers.gitlab.env_key="OPENAI_API_KEY"`,
				"--config", `model_providers.gitlab.wire_api="responses"`,
				"--config", `model_providers.gitlab.http_headers={"Authorization" = "Bearer token123"}`,
			},
		},
		{
			name: "multiple headers",
			headers: map[string]string{
				"Authorization": "Bearer token123",
				"X-Custom":      "value",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := createCodexConfigArgs(tc.headers)

			if len(tc.headers) <= 1 {
				assert.Equal(t, tc.expected, result)
			} else {
				// For multiple headers, just check the base args and that headers are included
				assert.Contains(t, result, "--config")
				assert.Contains(t, result, `model_provider="gitlab"`)
				assert.Contains(t, result, `model_providers.gitlab.name="GitLab Managed Codex"`)
				assert.Contains(t, result, `model_providers.gitlab.base_url="`+CloudConnectorUrl+`"`)
				assert.Contains(t, result, `model_providers.gitlab.env_key="OPENAI_API_KEY"`)
				assert.Contains(t, result, `model_providers.gitlab.wire_api="responses"`)

				// Check that headers are included in the result
				headerConfig := ""
				for _, arg := range result {
					if arg == "model_providers.gitlab.http_headers={\"Authorization\" = \"Bearer token123\", \"X-Custom\" = \"value\"}" ||
						arg == "model_providers.gitlab.http_headers={\"X-Custom\" = \"value\", \"Authorization\" = \"Bearer token123\"}" {
						headerConfig = arg
						break
					}
				}
				assert.NotEmpty(t, headerConfig, "Expected headers config not found")
			}
		})
	}
}

func TestCloudConnectorUrl(t *testing.T) {
	assert.Equal(t, "https://cloud.gitlab.com/ai/v1/proxy/openai/v1", CloudConnectorUrl)
}
