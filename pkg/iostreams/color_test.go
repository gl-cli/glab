package iostreams

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_isColorEnabled(t *testing.T) {
	preRun := func() {
		checkedNoColor = false // Reset it before each run
	}

	t.Run("default", func(t *testing.T) {
		preRun()

		got := isColorEnabled()
		assert.True(t, got)
	})

	t.Run("NO_COLOR", func(t *testing.T) {
		preRun()

		t.Setenv("NO_COLOR", "")

		got := isColorEnabled()
		assert.False(t, got)
	})

	t.Run("COLOR_ENABLED == 1", func(t *testing.T) {
		preRun()

		t.Setenv("NO_COLOR", "")
		t.Setenv("COLOR_ENABLED", "1")

		got := isColorEnabled()
		assert.True(t, got)
	})

	t.Run("COLOR_ENABLED == true", func(t *testing.T) {
		preRun()

		t.Setenv("NO_COLOR", "")
		t.Setenv("COLOR_ENABLED", "true")

		got := isColorEnabled()
		assert.True(t, got)
	})
}

func Test_makeColorFunc(t *testing.T) {
	tests := []struct {
		name         string
		color        string
		colorEnabled bool
		is256color   bool
		want         string
	}{
		{
			"gray",
			"black+h",
			true,
			false,
			"text",
		},
		{
			"gray_256",
			"black+h",
			true,
			true,
			fmt.Sprintf("\x1b[38;5;242m%s\x1b[m", "text"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Tests do not output to the "terminal" so they ignore colors in the output.
			// This setting needs to be forced for these tests to check colors properly.
			_isStdoutTerminal = true

			if tt.colorEnabled {
				t.Setenv("COLOR_ENABLED", "true")
			} else {
				t.Setenv("NO_COLOR", "true")
			}

			if tt.is256color {
				t.Setenv("TERM", "xterm-256color")
			}

			fn := makeColorFunc(tt.color)
			got := fn("text")

			require.Equal(t, tt.want, got)
		})
	}
}
