package codex

import (
	"fmt"
)

const (
	CloudConnectorUrl = "https://cloud.gitlab.com/ai/v1/proxy/openai/v1"
)

// CreateCodexConfigArgs builds Codex CLI --config args for GitLab integration
func createCodexConfigArgs(headers map[string]string) []string {
	args := []string{
		"--config", `model_provider="gitlab"`,
		"--config", `model_providers.gitlab.name="GitLab Managed Codex"`,
		"--config", `model_providers.gitlab.base_url="` + CloudConnectorUrl + `"`,
		"--config", `model_providers.gitlab.env_key="OPENAI_API_KEY"`,
		"--config", `model_providers.gitlab.wire_api="responses"`,
	}

	// Build inline TOML table for headers if any exist
	if len(headers) > 0 {
		headerStr := "{"
		first := true
		for k, v := range headers {
			if !first {
				headerStr += ", "
			}
			first = false
			headerStr += fmt.Sprintf("%q = %q", k, v)
		}
		headerStr += "}"
		args = append(args, "--config", fmt.Sprintf("model_providers.gitlab.http_headers=%s", headerStr))
	}

	return args
}
