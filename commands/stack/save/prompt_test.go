package save

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
)

func getMockEditor(input string, prompts *[]string) cmdutils.GetTextUsingEditor {
	return func(editor, tmpFileName, content string) (string, error) {
		*prompts = append(*prompts, content)
		return input, nil
	}
}

func Test_cleanDescription(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Commit message with only comments returns nothing",
			input:    "# Please\n# enter\n# the\n# commit\n# message\n",
			expected: "",
		},
		{
			name:     "Commit message with only comments and newlines returns nothing",
			input:    "\n\n# Test\n\n# Test\n\n",
			expected: "",
		},
		{
			name:     "Commit message with comments removes comments",
			input:    "Foo\n# Test\n# Test\n\n",
			expected: "Foo",
		},
		{
			name:     "Commit message preserves internal newlines",
			input:    "Implement feature X\n\nIt's nice\n\nBug: #1\n# Comment",
			expected: "Implement feature X\n\nIt's nice\n\nBug: #1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cleanDescription(tt.input)
			require.Equal(t, tt.expected, got)
		})
	}
}

func Test_promptForCommit(t *testing.T) {
	tests := []struct {
		name         string
		want         string
		wantErr      bool
		input        string
		defaultValue string
		noTTY        bool
	}{
		{
			name:  "A commit with hello returns hello",
			want:  "hello",
			input: "hello",
		},
		{
			name:    "A commit with no default and noTTY returns an error",
			input:   "hello",
			wantErr: true,
			noTTY:   true,
		},
		{
			name:         "A commit with noTTY and a default message returns the default message",
			input:        "hello",
			want:         "default message",
			defaultValue: "default message",
			noTTY:        true,
		},
		{
			name:         "A commit with noTTY and a default message returns the default message",
			input:        "",
			want:         "default message",
			defaultValue: "default message",
			noTTY:        true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isTTY := !tt.noTTY
			_, _, _, factory := setupTestFactory(nil, isTTY)
			prompts := []string{}
			getText := getMockEditor(tt.input, &prompts)
			got, err := promptForCommit(factory, getText, tt.defaultValue)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.Nil(t, err)
				require.Equal(t, tt.want, got)
			}
		})
	}
}
