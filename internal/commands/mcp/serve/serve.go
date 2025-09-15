package serve

import (
	"fmt"

	"gitlab.com/gitlab-org/cli/internal/mcpannotations"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
)

func NewCmdServe(_ cmdutils.Factory) *cobra.Command {
	serveCmd := &cobra.Command{
		Use:   "serve",
		Short: "Start MCP server with stdio transport",
		Long: heredoc.Doc(`
			Start a Model Context Protocol server that exposes GitLab functionality
			as tools for AI assistants like Claude Code.
			
			The server uses stdio transport for communication and provides tools for:

			- Managing issues (list, create, update, close, add notes)
			- Managing merge requests (list, create, update, merge, add notes)  
			- Managing projects (list, get details)
			- Managing CI/CD pipelines and jobs
			- And more GitLab functionality
			
			Configure this server in Claude Code by adding to your MCP settings:
			{
			  "mcpServers": {
			    "glab": {
			      "command": "glab",
			      "args": ["mcp", "serve"]
			    }
			  }
			}
		`),
		Example: heredoc.Doc(`
			$ glab mcp serve
		`),
		Annotations: map[string]string{
			mcpannotations.Safe: "true",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get the root command by traversing up the parent chain
			rootCmd := cmd
			for rootCmd.Parent() != nil {
				rootCmd = rootCmd.Parent()
			}

			// Initialize the MCP server
			server := newMCPServer(rootCmd)

			// Run the server (signal handling is done internally by server.ServeStdio)
			if err := server.Run(); err != nil {
				return fmt.Errorf("MCP server error: %w", err)
			}

			return nil
		},
	}

	return serveCmd
}
