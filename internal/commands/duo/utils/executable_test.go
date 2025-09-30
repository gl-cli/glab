package utils

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateClaudeExecutable(t *testing.T) {
	// Test when executable doesn't exist
	originalPath := os.Getenv("PATH")
	defer t.Setenv("PATH", originalPath)

	// Set PATH to empty to ensure claude is not found
	t.Setenv("PATH", "")

	err := ValidateExecutable("claude")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "claude executable not found in PATH")
}

func TestExtractClaudeArgs(t *testing.T) {
	tests := []struct {
		name          string
		osArgs        []string
		expectedArgs  []string
		expectedError string
	}{
		{
			name:         "claude with args",
			osArgs:       []string{"glab", "duo", "claude", "--help", "some", "args"},
			expectedArgs: []string{"--help", "some", "args"},
		},
		{
			name:         "claude without args",
			osArgs:       []string{"glab", "duo", "claude"},
			expectedArgs: []string{},
		},
		{
			name:          "no claude in args",
			osArgs:        []string{"glab", "duo", "ask", "something"},
			expectedError: "could not find 'claude' in command arguments",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Save original os.Args
			originalArgs := os.Args
			defer func() { os.Args = originalArgs }()

			// Set test args
			os.Args = tc.osArgs

			result, err := ExtractArgs("claude")

			if tc.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedArgs, result)
			}
		})
	}
}
