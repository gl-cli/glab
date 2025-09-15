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
			name:     "all fields present",
			cmd:      createMockCommand("full", "Short", "Long", "example content"),
			expected: "Short\n\nLong\n\nExamples:\nexample content",
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

	// Test child command picking up parent's help:arguments
	result := server.buildEnhancedDescription(child)
	expected := "Child command\n\nExamples:\nparent example\n\nArguments:\nparent help arguments"
	assert.Equal(t, expected, result)
}

// Tests for findNearestExample

func TestFindNearestExample(t *testing.T) {
	server := &mcpServer{}
	root, parent, child := createMockCommandHierarchy()

	tests := []struct {
		name     string
		cmd      *cobra.Command
		expected string
	}{
		{
			name:     "command with direct example",
			cmd:      parent,
			expected: "parent example",
		},
		{
			name:     "child inherits parent example",
			cmd:      child,
			expected: "parent example",
		},
		{
			name:     "root command example",
			cmd:      root,
			expected: "root example",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := server.findNearestExample(tt.cmd)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFindNearestExampleNoExample(t *testing.T) {
	server := &mcpServer{}
	cmd := createMockCommand("no-example", "Short", "", "")

	result := server.findNearestExample(cmd)
	assert.Equal(t, "", result)
}

// Tests for findNearestHelpArguments

func TestFindNearestHelpArguments(t *testing.T) {
	server := &mcpServer{}
	root, parent, child := createMockCommandHierarchy()

	tests := []struct {
		name     string
		cmd      *cobra.Command
		expected string
	}{
		{
			name:     "command with direct help:arguments",
			cmd:      parent,
			expected: "parent help arguments",
		},
		{
			name:     "child inherits parent help:arguments",
			cmd:      child,
			expected: "parent help arguments",
		},
		{
			name:     "no help:arguments found",
			cmd:      root,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := server.findNearestHelpArguments(tt.cmd)
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
		hasDefault   bool
		hasMin       bool
	}{
		{
			flagName:     "verbose",
			expectedType: "boolean",
			hasDefault:   false,
			hasMin:       false,
		},
		{
			flagName:     "output",
			expectedType: "string",
			hasDefault:   true,
			hasMin:       false,
		},
		{
			flagName:     "count",
			expectedType: "number",
			hasDefault:   true,
			hasMin:       false,
		},
		{
			flagName:     "labels",
			expectedType: "array",
			hasDefault:   true, // StringSlice flags get default "[]"
			hasMin:       false,
		},
		{
			flagName:     "port",
			expectedType: "number",
			hasDefault:   true,
			hasMin:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.flagName, func(t *testing.T) {
			flag := cmd.Flags().Lookup(tt.flagName)
			require.NotNil(t, flag)

			schema := server.buildFlagSchema(flag)
			require.NotNil(t, schema)

			assert.Equal(t, tt.expectedType, schema["type"])

			if tt.hasDefault {
				assert.Contains(t, schema, "default")
			} else {
				assert.NotContains(t, schema, "default")
			}

			if tt.hasMin {
				assert.Contains(t, schema, "minimum")
				assert.Equal(t, 0, schema["minimum"])
			} else {
				assert.NotContains(t, schema, "minimum")
			}

			// All should have description
			assert.Contains(t, schema, "description")
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
