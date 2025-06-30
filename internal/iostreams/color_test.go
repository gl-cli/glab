package iostreams

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_isColorEnabled(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		got := detectIsColorEnabled()
		assert.True(t, got)
	})

	t.Run("NO_COLOR", func(t *testing.T) {
		t.Setenv("NO_COLOR", "")

		got := detectIsColorEnabled()
		assert.False(t, got)
	})

	t.Run("COLOR_ENABLED == 1", func(t *testing.T) {
		t.Setenv("NO_COLOR", "")
		t.Setenv("COLOR_ENABLED", "1")

		got := detectIsColorEnabled()
		assert.True(t, got)
	})

	t.Run("COLOR_ENABLED == true", func(t *testing.T) {
		t.Setenv("NO_COLOR", "")
		t.Setenv("COLOR_ENABLED", "true")

		got := detectIsColorEnabled()
		assert.True(t, got)
	})
}

func Test_makeColorFunc(t *testing.T) {
	tests := []struct {
		name         string
		color        string
		colorEnabled bool
		term         string
		want         string
	}{
		{
			name:         "gray 16 colors",
			color:        "black+h",
			colorEnabled: true,
			term:         "xterm-16color",
			want:         "\x1b[0;90mtext\x1b[0m",
		},
		{
			name:         "gray 256 colors",
			color:        "black+h",
			colorEnabled: true,
			term:         "xterm-256color",
			want:         "\x1b[38;5;242mtext\x1b[m",
		},
		{
			name:         "no colors",
			color:        "black+h",
			colorEnabled: false,
			term:         "",
			want:         "text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("COLORTERM", "")
			t.Setenv("TERM", tt.term)

			fn := makeColorFunc(tt.colorEnabled, tt.color)
			got := fn("text")

			require.Equal(t, tt.want, got)
		})
	}
}
