package urlwrapper

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMDWrap(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "bare URL gets wrapped",
			input:    "Visit https://gitlab.com/docs for documentation",
			expected: "Visit [https://gitlab.com/docs](https://gitlab.com/docs) for documentation",
		},
		{
			name:     "already wrapped URL is preserved",
			input:    "See [documentation](https://gitlab.com/docs) here",
			expected: "See [documentation](https://gitlab.com/docs) here",
		},
		{
			name:     "URL inside single backticks is preserved",
			input:    "If unset, defaults to `https://gitlab.com`.",
			expected: "If unset, defaults to `https://gitlab.com`.",
		},
		{
			name:     "URL inside double backticks is preserved",
			input:    "Use ``code with https://example.com inside``",
			expected: "Use ``code with https://example.com inside``",
		},
		{
			name:     "multiple URLs with mixed contexts",
			input:    "See `https://gitlab.com` or visit https://github.com for more",
			expected: "See `https://gitlab.com` or visit [https://github.com](https://github.com) for more",
		},
		{
			name:     "duplicate URLs with different contexts",
			input:    "See https://gitlab.com for docs. The default is `https://gitlab.com` for most users.",
			expected: "See [https://gitlab.com](https://gitlab.com) for docs. The default is `https://gitlab.com` for most users.",
		},
		{
			name:     "same URL appears multiple times outside backticks",
			input:    "Visit https://example.com or https://example.com again",
			expected: "Visit [https://example.com](https://example.com) or [https://example.com](https://example.com) again",
		},
		{
			name:     "unmatched backtick does not affect URL wrapping",
			input:    "This has an unmatched ` backtick and https://example.com should be wrapped",
			expected: "This has an unmatched ` backtick and [https://example.com](https://example.com) should be wrapped",
		},
		{
			name:     "escaped backtick (odd backslashes)",
			input:    "Use \\` for literal backtick and https://example.com for docs",
			expected: "Use \\` for literal backtick and [https://example.com](https://example.com) for docs",
		},
		{
			name:     "escaped backslash before backtick (even backslashes)",
			input:    "Use \\\\`code` for literal backslash, then https://example.com",
			expected: "Use \\\\`code` for literal backslash, then [https://example.com](https://example.com)",
		},
		{
			name:     "URL inside code after escaped backslash",
			input:    "Text \\\\`https://example.com` should not wrap the URL",
			expected: "Text \\\\`https://example.com` should not wrap the URL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MDWrap(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
