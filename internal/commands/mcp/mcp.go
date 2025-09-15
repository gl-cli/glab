package mcp

import (
	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	mcpServeCmd "gitlab.com/gitlab-org/cli/internal/commands/mcp/serve"
)

func NewCmdMCP(f cmdutils.Factory) *cobra.Command {
	mcpCmd := &cobra.Command{
		Use:   "mcp <command>",
		Short: "Work with Model Context Protocol (MCP) server",
		Long: heredoc.Doc(`
			Manage Model Context Protocol server functionality for GitLab integration.
			
			The MCP server exposes GitLab functionality as tools that can be used by
			AI assistants like Claude Code to interact with GitLab projects, issues,
			merge requests, pipelines, and other resources.
		`),
		Example: heredoc.Doc(`
			$ glab mcp serve
		`),
	}

	mcpCmd.AddCommand(mcpServeCmd.NewCmdServe(f))

	return mcpCmd
}
