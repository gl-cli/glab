package serve

import (
	"context"
	"fmt"
	"iter"
	"os"
	"os/exec"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"gitlab.com/gitlab-org/cli/internal/mcpannotations"
)

const (
	// Parameter names for the nested MCP tool structure
	argsParam   = "args"
	flagsParam  = "flags"
	limitParam  = "limit"
	offsetParam = "offset"

	// Default response limit in runes (balances usefulness vs token consumption)
	defaultResponseLimit = 50000
)

// mcpServer wraps the MCP server with GitLab client access
type mcpServer struct {
	server  *server.MCPServer
	rootCmd *cobra.Command
}

// newMCPServer creates a new MCP server instance using mark3labs/mcp-go
func newMCPServer(rootCmd *cobra.Command) *mcpServer {
	// Create MCP server with usage instructions
	instructions := `GitLab CLI MCP Server - Provides access to GitLab functionality through glab commands.

General Usage:
- Use --help flag with any tool to get detailed usage information
- For large outputs, use limit/offset parameters for pagination
- Check 'total_size' in response metadata to navigate results
- Most tools support common flags like --output for formatting`

	mcpSrv := server.NewMCPServer(
		"glab-mcp-server",
		"1.0.0",
		server.WithToolCapabilities(true),
		server.WithInstructions(instructions),
	)

	glabServer := &mcpServer{
		server:  mcpSrv,
		rootCmd: rootCmd,
	}

	// Register all GitLab tools dynamically
	glabServer.registerToolsFromCommands()

	return glabServer
}

// Run starts the MCP server with stdio transport
func (s *mcpServer) Run() error {
	return server.ServeStdio(s.server)
}

// registerToolsFromCommands automatically registers all glab commands as MCP tools
func (s *mcpServer) registerToolsFromCommands() {
	for cmd, path := range s.iterCommands(s.rootCmd, []string{}) {
		// Only register leaf commands that have RunE and are not the root command
		if cmd.RunE == nil || cmd == s.rootCmd {
			continue
		}

		toolName := "glab_" + strings.Join(path, "_")
		description := s.buildEnhancedDescription(cmd)
		if description == "" {
			description = fmt.Sprintf("Execute glab %s command", strings.Join(path, " "))
		}

		// Build the tool with dynamic schema using the builder pattern
		tool := s.buildToolFromCommand(toolName, description, cmd)

		// Create handler for this command
		handler := s.createCommandHandler(path, cmd)

		// Register the tool
		s.server.AddTool(tool, handler)
	}
}

func (s *mcpServer) iterCommands(cmd *cobra.Command, path []string) iter.Seq2[*cobra.Command, []string] {
	return func(yield func(*cobra.Command, []string) bool) {
		cmdName := strings.Fields(cmd.Use)[0]

		// Skip root "glab" command from path - remove binary name earlier
		var currentPath []string
		if len(path) == 0 && cmdName == "glab" {
			// This is the root command, start with empty path
			currentPath = []string{}
		} else {
			currentPath = append(path, cmdName)
		}

		// Process current command
		if !yield(cmd, currentPath) {
			return
		}

		// Recursively process subcommands
		for _, subCmd := range cmd.Commands() {
			for c, p := range s.iterCommands(subCmd, currentPath) {
				if !yield(c, p) {
					return
				}
			}
		}
	}
}

// buildEnhancedDescription creates an optimized description with truncated content and standard guidance
func (s *mcpServer) buildEnhancedDescription(cmd *cobra.Command) string {
	var parts []string

	// Start with the command's short description
	if cmd.Short != "" {
		parts = append(parts, cmd.Short)
	}

	// Add truncated long description if present
	if cmd.Long != "" {
		truncatedLong := s.truncateAtWordBoundary(cmd.Long, 100)
		parts = append(parts, "", truncatedLong)
	}

	// Add standard guidance
	description := strings.Join(parts, "\n")
	return s.addStandardGuidance(description)
}

// truncateAtWordBoundary truncates text to maxChars at the nearest word boundary
func (s *mcpServer) truncateAtWordBoundary(text string, maxChars int) string {
	if len(text) <= maxChars {
		return text
	}

	// Find the last space within the limit, accounting for "..." suffix
	for i := maxChars - 4; i >= 0; i-- {
		if text[i] == ' ' || text[i] == '\n' {
			return strings.TrimSpace(text[:i]) + "..."
		}
	}

	// If no space found, hard truncate accounting for "..." suffix
	return text[:maxChars-3] + "..."
}

// addStandardGuidance is no longer needed since guidance is provided at server level
// Keeping this as a no-op for now in case we want tool-specific guidance later
func (s *mcpServer) addStandardGuidance(description string) string {
	return description
}

// buildToolFromCommand creates a tool using the builder pattern with dynamic schema
func (s *mcpServer) buildToolFromCommand(toolName, description string, cmd *cobra.Command) mcp.Tool {
	// Start building the tool
	toolOptions := []mcp.ToolOption{
		mcp.WithDescription(description),
	}

	// Determine if this is a destructive command
	isDestructive := s.isDestructiveCommand(cmd)
	toolOptions = append(toolOptions, mcp.WithDestructiveHintAnnotation(isDestructive))

	// Create nested flags object schema with all available flags
	flagsProperties := make(map[string]any)

	// Add parameters for each flag to the flags object
	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		if !flag.Hidden && flag.Name != "help" {
			flagName := strings.ReplaceAll(flag.Name, "-", "_")
			flagSchema := s.buildFlagSchema(flag)
			if flagSchema != nil {
				flagsProperties[flagName] = flagSchema
			}
		}
	})

	// Add persistent flags from parent commands to the flags object
	cmd.PersistentFlags().VisitAll(func(flag *pflag.Flag) {
		if !flag.Hidden && flag.Name != "help" {
			flagName := strings.ReplaceAll(flag.Name, "-", "_")

			// Check if we already added this flag from local flags
			alreadyAdded := false
			cmd.Flags().VisitAll(func(localFlag *pflag.Flag) {
				if localFlag.Name == flag.Name {
					alreadyAdded = true
				}
			})

			if !alreadyAdded {
				flagSchema := s.buildFlagSchema(flag)
				if flagSchema != nil {
					flagsProperties[flagName] = flagSchema
				}
			}
		}
	})

	// Add the new nested structure parameters
	toolOptions = append(toolOptions,
		// Simplified parameter definitions to reduce token usage
		mcp.WithArray(argsParam, mcp.WithStringItems(), mcp.Description("Positional arguments")),
		mcp.WithObject(flagsParam, mcp.Properties(flagsProperties), mcp.Description("Command flags")),
		mcp.WithNumber(limitParam, mcp.Description("Response size limit"), mcp.DefaultNumber(float64(defaultResponseLimit))),
		mcp.WithNumber(offsetParam, mcp.Description("Pagination offset"), mcp.DefaultNumber(0)),
	)

	return mcp.NewTool(toolName, toolOptions...)
}

// buildFlagSchema creates a JSON schema object for a flag (used in nested flags object)
func (s *mcpServer) buildFlagSchema(flag *pflag.Flag) map[string]any {
	flagType := flag.Value.Type()
	schema := make(map[string]any)

	// Removed descriptions and defaults to minimize token usage
	// LLMs can infer flag purpose from flag names

	// Minimal type information only
	switch flagType {
	case "bool":
		schema["type"] = "boolean"
	case "stringSlice", "stringArray", "intSlice":
		schema["type"] = "array"
	case "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64", "float32", "float64":
		schema["type"] = "number"
	default:
		schema["type"] = "string"
	}

	return schema
}

// createCommandHandler creates a handler function for a specific glab command
func (s *mcpServer) createCommandHandler(cmdPath []string, cmd *cobra.Command) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Get parameters from the request
		params := request.GetArguments()

		// Convert MCP parameters to command line arguments and extract response config
		args, config := s.convertParamsToArgs(params, cmd)

		// Execute the glab command
		output, err := s.executeGlabCommand(cmdPath, args)
		if err != nil {
			// Return the error as content so the user can see what went wrong
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: output, // This includes the actual error message from the command
					},
				},
				IsError: true, // Mark this as an error response
			}, nil
		}

		// Process output with rune-based limiting and metadata
		processedOutput, metadata := s.processOutput(output, config)

		// Return the result with clean content and structured metadata
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: processedOutput,
				},
			},
			StructuredContent: map[string]any{
				"pagination": metadata,
			},
		}, nil
	}
}

// responseConfig holds output processing configuration
type responseConfig struct {
	Limit  int
	Offset int
}

// processOutput handles rune-based output limiting and generates metadata
func (s *mcpServer) processOutput(output string, config responseConfig) (string, map[string]any) {
	// Convert to runes for Unicode-safe processing
	runes := []rune(output)
	totalSize := len(runes)

	// Calculate slice bounds with support for negative offsets (counting from end)
	start := config.Offset
	if start < 0 {
		// Negative offset: count from the end like 'tail'
		start = max(totalSize+start, 0)
	}
	if start > totalSize {
		start = totalSize
	}

	end := min(start+config.Limit, totalSize)

	// Extract the slice
	var processedRunes []rune
	if start < totalSize {
		processedRunes = runes[start:end]
	}

	// Create comprehensive metadata
	truncated := (start > 0 || end < totalSize)
	metadata := map[string]any{
		"total_size":   totalSize,
		"limit":        config.Limit,
		"offset":       config.Offset,
		"actual_start": start,
		"actual_end":   end,
		"actual_size":  len(processedRunes),
		"truncated":    truncated,
	}

	// Add helpful navigation hints for AI
	if truncated {
		metadata["navigation_hints"] = map[string]any{
			"to_beginning": 0,
			"to_end":       totalSize - config.Limit,
			"next_page":    end,
			"prev_page":    start - config.Limit,
		}
		metadata["usage_guide"] = "To navigate: use 'to_end' offset to jump to end where failures typically occur, 'next_page' for next section, or calculate custom offset. Example: for logs that end with errors, use offset = total_size - limit."
	}

	return string(processedRunes), metadata
}

// convertParamsToArgs converts MCP JSON parameters to command line arguments and extracts response config
func (s *mcpServer) convertParamsToArgs(params map[string]any, cmd *cobra.Command) ([]string, responseConfig) {
	var args []string
	var positionals []string
	config := responseConfig{
		Limit:  defaultResponseLimit,
		Offset: 0,
	}

	// Handle args (positional arguments)
	if argsParam, exists := params[argsParam]; exists {
		if argArray, ok := argsParam.([]any); ok {
			for _, arg := range argArray {
				if str, ok := arg.(string); ok && str != "" {
					positionals = append(positionals, str)
				}
			}
		}
	}

	// Handle limit parameter
	if limitParam, exists := params[limitParam]; exists {
		if f64, ok := limitParam.(float64); ok {
			config.Limit = int(f64)
		}
	}

	// Handle offset parameter
	if offsetParam, exists := params[offsetParam]; exists {
		if f64, ok := offsetParam.(float64); ok {
			config.Offset = int(f64)
		}
	}

	// Handle flags object
	if flagsParam, exists := params[flagsParam]; exists {
		if flagsObj, ok := flagsParam.(map[string]any); ok {
			for key, value := range flagsObj {
				if value == nil {
					continue
				}

				// Convert snake_case to kebab-case for CLI flags
				flagName := strings.ReplaceAll(key, "_", "-")

				// Check if this is a known flag
				flag := cmd.Flags().Lookup(flagName)
				if flag == nil {
					// Try original key name
					flag = cmd.Flags().Lookup(key)
				}

				// Process the parameter value
				switch v := value.(type) {
				case bool:
					if v && flag != nil {
						args = append(args, "--"+flagName)
					}
				case string:
					if v != "" {
						if flag != nil {
							args = append(args, "--"+flagName, v)
						}
					}
				case []any:
					// Handle arrays (like labels)
					for _, item := range v {
						if str, ok := item.(string); ok && str != "" {
							args = append(args, "--"+flagName, str)
						}
					}
				case float64:
					// Handle numbers from JSON
					if v != 0 {
						// For large integers (like pipeline IDs), format without decimals and avoid scientific notation
						var numStr string
						if v == float64(int64(v)) {
							// This is an integer value, format as int to avoid precision issues
							numStr = fmt.Sprintf("%d", int64(v))
						} else {
							// This is a float value
							numStr = fmt.Sprintf("%g", v)
						}

						if flag != nil {
							args = append(args, "--"+flagName, numStr)
						}
					}
				default:
					// Convert other types to string
					if str := fmt.Sprintf("%v", value); str != "" && str != "0" && str != "false" {
						if flag != nil {
							args = append(args, "--"+flagName, str)
						}
					}
				}
			}
		}
	}

	// Add positional arguments at the end
	args = append(args, positionals...)

	return args, config
}

// executeGlabCommand executes a glab command and captures its output
func (s *mcpServer) executeGlabCommand(cmdPath []string, args []string) (string, error) {
	// Get the current binary (same one running MCP server)
	currentBinary, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to get current executable: %w", err)
	}

	// Build full command arguments
	fullArgs := append(cmdPath, args...)

	// Execute subprocess
	cmd := exec.Command(currentBinary, fullArgs...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		// On failure, return the output (which includes stderr) with the error
		return string(output), err
	}

	// On success, return stdout content
	return string(output), nil
}

// isDestructiveCommand determines if a command is destructive based on annotations
func (s *mcpServer) isDestructiveCommand(cmd *cobra.Command) bool {
	// All executable commands should have annotations
	if cmd.Annotations != nil {
		if val, exists := cmd.Annotations[mcpannotations.Destructive]; exists {
			return val == "true"
		}
		if val, exists := cmd.Annotations[mcpannotations.Safe]; exists {
			return val != "true"
		}
	}

	// Default to destructive for safety if no annotation found (should not happen for executable commands)
	return true
}
