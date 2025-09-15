// Package claude provides commands for integrating with Claude Code through GitLab Duo.
package claude

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"gitlab.com/gitlab-org/cli/internal/mcpannotations"

	"gitlab.com/gitlab-org/cli/internal/api"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/internal/iostreams"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
)

// opts holds the configuration options for Claude commands.
type opts struct {
	Prompt    string
	IO        *iostreams.IOStreams
	apiClient func(repoHost string) (*api.Client, error)
	BaseRepo  func() (glrepo.Interface, error)
}

func NewCmdClaude(f cmdutils.Factory) *cobra.Command {
	opts := &opts{
		IO:        f.IO(),
		apiClient: f.ApiClient,
		BaseRepo:  f.BaseRepo,
	}

	duoClaudeCmd := &cobra.Command{
		Use:   "claude [flags] [args]",
		Short: "Launch Claude Code with GitLab Duo integration",
		Long: heredoc.Doc(`
			Launch Claude Code with automatic GitLab authentication, proxy configuration,
			and GitLab MCP tools integration. All flags and arguments are passed through 
			to the Claude executable.
			
			This command automatically configures Claude Code to work with GitLab AI services,
			handling authentication tokens and API endpoints based on your current repository.
			It also provides access to all GitLab functionality through MCP tools, allowing
			you to interact with issues, merge requests, CI/CD pipelines, and more directly
			from within Claude Code.
		`),
		Example: heredoc.Doc(`
			$ glab duo claude
			$ glab duo claude -p "List all open issues in this project"
			$ glab duo claude -p "Write a function to calculate Fibonacci numbers"
		`),
		// Allow unknown flags to be passed through to the claude command
		FParseErrWhitelist: cobra.FParseErrWhitelist{
			UnknownFlags: true,
		},
		Annotations: map[string]string{
			mcpannotations.Destructive: "true",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// Fetch repo host
			var repoHost string
			if baseRepo, err := opts.BaseRepo(); err == nil {
				repoHost = baseRepo.RepoHost()
			}

			// Get API client
			c, err := opts.apiClient(repoHost)
			if err != nil {
				return err
			}

			// Fetch direct_access token
			token, err := fetchDirectAccessToken(c.Lab())
			if err != nil {
				return fmt.Errorf("failed to retrieve GitLab Duo access token: %w", err)
			}

			// Validate Claude executable exists
			if err := validateClaudeExecutable(); err != nil {
				return fmt.Errorf("claude executable validation failed: %w", err)
			}

			wasAbleToSetApiKeyHelper := setClaudeSettings()

			// Extract Claude command arguments
			claudeArgs, err := extractClaudeArgs()
			if err != nil {
				return fmt.Errorf("failed to parse command arguments: %w", err)
			}

			// Add ephemeral MCP configuration for glab server using current binary
			currentBinary, err := os.Executable()
			if err != nil {
				return fmt.Errorf("failed to determine current executable path: %w", err)
			}

			// Create MCP configuration struct and marshal to JSON
			mcpConfigStruct := map[string]interface{}{
				"mcpServers": map[string]interface{}{
					"glab": map[string]interface{}{
						"command": currentBinary,
						"args":    []string{"mcp", "serve"},
					},
				},
			}

			mcpConfigBytes, err := json.Marshal(mcpConfigStruct)
			if err != nil {
				return fmt.Errorf("failed to marshal MCP config: %w", err)
			}

			claudeArgs = append(claudeArgs, "--mcp-config", string(mcpConfigBytes))

			// Execute Claude command with all arguments
			claudeCmd := exec.Command(ClaudeExecutable, claudeArgs...)

			// Connect standard input/output/error
			claudeCmd.Stdin = opts.IO.In
			claudeCmd.Stdout = opts.IO.StdOut
			claudeCmd.Stderr = opts.IO.StdErr

			// Set environment variables for the Claude command
			claudeCmd.Env = append(os.Environ(),
				fmt.Sprintf("%s=%s", EnvAnthropicCustomHeaders, getHeaderEnv(token.Headers)),
				fmt.Sprintf("%s=%s", EnvAnthropicBaseURL, CloudConnectorUrl),
				fmt.Sprintf("%s=%s", EnvAnthropicModel, DefaultClaudeModel),
			)

			if !wasAbleToSetApiKeyHelper {
				claudeCmd.Env = append(claudeCmd.Env, fmt.Sprintf("%s=%s", EnvAnthropicAuthToken, token.Token))
			}

			// Execute the command
			if err := claudeCmd.Run(); err != nil {
				return fmt.Errorf("failed to execute Claude Code: %w", err)
			}

			return nil
		},
	}

	duoClaudeCmd.AddCommand(NewCmdToken(f))

	return duoClaudeCmd
}
