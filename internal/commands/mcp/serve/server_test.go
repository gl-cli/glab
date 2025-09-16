package serve

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cli/internal/mcpannotations"
)

// Mock command creation helpers

func createMockCommand(name, short, long, example string) *cobra.Command {
	cmd := &cobra.Command{
		Use:     name,
		Short:   short,
		Long:    long,
		Example: example,
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}
	return cmd
}

func createMockCommandWithAnnotations(name, short string, annotations map[string]string) *cobra.Command {
	cmd := &cobra.Command{
		Use:         name,
		Short:       short,
		Annotations: annotations,
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}
	return cmd
}

func createMockCommandHierarchy() (*cobra.Command, *cobra.Command, *cobra.Command) {
	root := createMockCommand("root", "Root command", "Root long description", "root example")
	parent := createMockCommand("parent", "Parent command", "Parent long description", "parent example")
	parent.Annotations = map[string]string{
		"help:arguments": "parent help arguments",
	}
	child := createMockCommand("child", "Child command", "", "")

	root.AddCommand(parent)
	parent.AddCommand(child)

	return root, parent, child
}

func createMockCommandWithFlags() *cobra.Command {
	cmd := createMockCommand("test", "Test command", "", "")

	// Add various flag types
	cmd.Flags().Bool("verbose", false, "Enable verbose output")
	cmd.Flags().String("output", "text", "Output format")
	cmd.Flags().Int("count", 10, "Number of items")
	cmd.Flags().StringSlice("labels", []string{}, "List of labels")
	cmd.Flags().Uint("port", 8080, "Port number")

	return cmd
}

// Tests for buildEnhancedDescription

func TestBuildEnhancedDescription(t *testing.T) {
	server := &mcpServer{}

	tests := []struct {
		name     string
		cmd      *cobra.Command
		expected string
	}{
		{
			name:     "empty command",
			cmd:      createMockCommand("empty", "", "", ""),
			expected: "",
		},
		{
			name:     "short description only",
			cmd:      createMockCommand("short", "Short description", "", ""),
			expected: "Short description",
		},
		{
			name:     "short and long descriptions",
			cmd:      createMockCommand("both", "Short desc", "Long description here", ""),
			expected: "Short desc\n\nLong description here",
		},
		{
			name:     "long description truncation",
			cmd:      createMockCommand("truncated", "Short", "This is a very long description that should be truncated at one hundred characters because it exceeds the limit", ""),
			expected: "Short\n\nThis is a very long description that should be truncated at one hundred characters because it...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := server.buildEnhancedDescription(tt.cmd)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildEnhancedDescriptionWithHierarchy(t *testing.T) {
	server := &mcpServer{}
	root, _, child := createMockCommandHierarchy()
	server.rootCmd = root

	// Test child command with simplified format
	result := server.buildEnhancedDescription(child)
	expected := "Child command"
	assert.Equal(t, expected, result)
}

// Tests for truncateAtWordBoundary

func TestTruncateAtWordBoundary(t *testing.T) {
	server := &mcpServer{}

	tests := []struct {
		name     string
		text     string
		maxChars int
		expected string
	}{
		{
			name:     "short text no truncation",
			text:     "Short text",
			maxChars: 20,
			expected: "Short text",
		},
		{
			name:     "truncate at word boundary",
			text:     "This is a long text that should be truncated",
			maxChars: 20,
			expected: "This is a long...",
		},
		{
			name:     "hard truncate if no spaces",
			text:     "Verylongtextwithnospaces",
			maxChars: 10,
			expected: "Verylon...",
		},
		{
			name:     "truncate at newline",
			text:     "First line\nSecond line that is longer",
			maxChars: 15,
			expected: "First line...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := server.truncateAtWordBoundary(tt.text, tt.maxChars)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Tests for addStandardGuidance

func TestAddStandardGuidance(t *testing.T) {
	server := &mcpServer{}

	tests := []struct {
		name        string
		description string
		expected    string
	}{
		{
			name:        "empty description",
			description: "",
			expected:    "",
		},
		{
			name:        "add guidance to description",
			description: "Command description",
			expected:    "Command description",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := server.addStandardGuidance(tt.description)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Tests for buildFlagSchema

func TestBuildFlagSchema(t *testing.T) {
	server := &mcpServer{}
	cmd := createMockCommandWithFlags()

	tests := []struct {
		flagName     string
		expectedType string
	}{
		{
			flagName:     "verbose",
			expectedType: "boolean",
		},
		{
			flagName:     "output",
			expectedType: "string",
		},
		{
			flagName:     "count",
			expectedType: "number",
		},
		{
			flagName:     "labels",
			expectedType: "array",
		},
		{
			flagName:     "port",
			expectedType: "number",
		},
	}

	for _, tt := range tests {
		t.Run(tt.flagName, func(t *testing.T) {
			flag := cmd.Flags().Lookup(tt.flagName)
			require.NotNil(t, flag)

			schema := server.buildFlagSchema(flag)
			require.NotNil(t, schema)

			assert.Equal(t, tt.expectedType, schema["type"])

			// Minimal schema only contains type
			assert.NotContains(t, schema, "default")
			assert.NotContains(t, schema, "description")
			assert.NotContains(t, schema, "minimum")
		})
	}
}

// Tests for isDestructiveCommand

func TestIsDestructiveCommand(t *testing.T) {
	server := &mcpServer{}

	tests := []struct {
		name        string
		annotations map[string]string
		expected    bool
	}{
		{
			name:        "no annotations - defaults to destructive",
			annotations: nil,
			expected:    true,
		},
		{
			name:        "explicitly safe",
			annotations: map[string]string{mcpannotations.Safe: "true"},
			expected:    false,
		},
		{
			name:        "explicitly destructive",
			annotations: map[string]string{mcpannotations.Destructive: "true"},
			expected:    true,
		},
		{
			name:        "safe annotation false",
			annotations: map[string]string{mcpannotations.Safe: "false"},
			expected:    true,
		},
		{
			name:        "destructive annotation false",
			annotations: map[string]string{mcpannotations.Destructive: "false"},
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := createMockCommandWithAnnotations("test", "Test", tt.annotations)
			result := server.isDestructiveCommand(cmd)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Tests for convertParamsToArgs

func TestConvertParamsToArgs(t *testing.T) {
	server := &mcpServer{}
	cmd := createMockCommandWithFlags()

	tests := []struct {
		name     string
		params   map[string]any
		expected []string
	}{
		{
			name:     "empty params",
			params:   map[string]any{},
			expected: []string{},
		},
		{
			name: "boolean flag true",
			params: map[string]any{
				"flags": map[string]any{
					"verbose": true,
				},
			},
			expected: []string{"--verbose"},
		},
		{
			name: "boolean flag false",
			params: map[string]any{
				"flags": map[string]any{
					"verbose": false,
				},
			},
			expected: []string{},
		},
		{
			name: "string flag",
			params: map[string]any{
				"flags": map[string]any{
					"output": "json",
				},
			},
			expected: []string{"--output", "json"},
		},
		{
			name: "number flag",
			params: map[string]any{
				"flags": map[string]any{
					"count": float64(25),
				},
			},
			expected: []string{"--count", "25"},
		},
		{
			name: "array flag",
			params: map[string]any{
				"flags": map[string]any{
					"labels": []any{"bug", "urgent"},
				},
			},
			expected: []string{"--labels", "bug", "--labels", "urgent"},
		},
		{
			name: "positional args",
			params: map[string]any{
				"args": []any{"arg1", "arg2"},
			},
			expected: []string{"arg1", "arg2"},
		},
		{
			name: "mixed params",
			params: map[string]any{
				"args": []any{"pos1"},
				"flags": map[string]any{
					"verbose": true,
					"output":  "json",
				},
			},
			expected: []string{"--verbose", "--output", "json", "pos1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args, _ := server.convertParamsToArgs(tt.params, cmd)
			assert.ElementsMatch(t, tt.expected, args)
		})
	}
}

// Tests for processOutput

func TestProcessOutput(t *testing.T) {
	server := &mcpServer{}

	tests := []struct {
		name           string
		output         string
		config         responseConfig
		expectedText   string
		expectedLength int
		truncated      bool
	}{
		{
			name:           "short output no limiting",
			output:         "hello world",
			config:         responseConfig{Limit: 100, Offset: 0},
			expectedText:   "hello world",
			expectedLength: 11,
			truncated:      false,
		},
		{
			name:           "output with limiting",
			output:         "hello world",
			config:         responseConfig{Limit: 5, Offset: 0},
			expectedText:   "hello",
			expectedLength: 5,
			truncated:      true,
		},
		{
			name:           "output with offset",
			output:         "hello world",
			config:         responseConfig{Limit: 5, Offset: 6},
			expectedText:   "world",
			expectedLength: 5,
			truncated:      true,
		},
		{
			name:           "negative offset",
			output:         "hello world",
			config:         responseConfig{Limit: 5, Offset: -5},
			expectedText:   "world",
			expectedLength: 5,
			truncated:      true,
		},
		{
			name:           "unicode handling",
			output:         "héllo wörld",
			config:         responseConfig{Limit: 5, Offset: 0},
			expectedText:   "héllo",
			expectedLength: 5,
			truncated:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, metadata := server.processOutput(tt.output, tt.config)

			assert.Equal(t, tt.expectedText, result)
			assert.Equal(t, tt.expectedLength, len([]rune(result)))
			assert.Equal(t, tt.truncated, metadata["truncated"])
			assert.Equal(t, len([]rune(tt.output)), metadata["total_size"])
			assert.Equal(t, tt.config.Limit, metadata["limit"])
			assert.Equal(t, tt.config.Offset, metadata["offset"])
		})
	}
}
