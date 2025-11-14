package mcp

import (
	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"

	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	mcpServeCmd "gitlab.com/gitlab-org/cli/internal/commands/mcp/serve"
	"gitlab.com/gitlab-org/cli/internal/text"
)

func NewCmdMCP(f cmdutils.Factory) *cobra.Command {
	mcpCmd := &cobra.Command{
		Use:   "mcp <command>",
		Short: "Work with a Model Context Protocol (MCP) server. (EXPERIMENTAL)",
		Long: heredoc.Doc(`
			Manage Model Context Protocol server features for GitLab integration.

			The MCP server exposes GitLab features as tools for use by
			AI assistants (like Claude Code) to interact with GitLab projects, issues,
			merge requests, pipelines, and other resources.
		`) + text.ExperimentalString,
		Example: heredoc.Doc(`
			$ glab mcp serve
		`),
	}

	mcpCmd.AddCommand(mcpServeCmd.NewCmdServe(f))

	return mcpCmd
}
