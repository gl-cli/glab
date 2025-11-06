//go:build !integration

package api

import "testing"

func TestIsTokenConfigured(t *testing.T) {
	tests := []struct {
		name  string
		token string
		want  bool
	}{
		{
			name:  "Non-empty token",
			token: "some-token",
			want:  true,
		},
		{
			name:  "Empty token",
			token: "",
			want:  false,
		},
		{
			name:  "Whitespace token",
			token: "   ",
			want:  false, // Should be false since whitespace-only tokens are not valid
		},
		{
			name:  "Tab and newline token",
			token: "\t\n  \r",
			want:  false, // Should be false since whitespace-only tokens are not valid
		},
		{
			name:  "Token with leading/trailing whitespace",
			token: "  valid-token  ",
			want:  true, // Should be true since it contains non-whitespace content
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsTokenConfigured(tt.token); got != tt.want {
				t.Errorf("IsTokenConfigured() = %v, want %v", got, tt.want)
			}
		})
	}
}
