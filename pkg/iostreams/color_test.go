package iostreams

import (
	"fmt"
	"os"
	"testing"

	"github.com/alecthomas/assert"
	"github.com/stretchr/testify/require"
)

func Test_isColorEnabled(t *testing.T) {
	preRun := func() {
		os.Unsetenv("NO_COLOR")
		os.Unsetenv("COLOR_ENABLED")
		checkedNoColor = false // Reset it before each run
	}

	t.Run("default", func(t *testing.T) {
		preRun()

		got := isColorEnabled()
		assert.True(t, got)
	})

	t.Run("NO_COLOR", func(t *testing.T) {
		preRun()

		_ = os.Setenv("NO_COLOR", "")

		got := isColorEnabled()
		assert.False(t, got)
	})

	t.Run("COLOR_ENABLED == 1", func(t *testing.T) {
		preRun()

		_ = os.Setenv("NO_COLOR", "")
		_ = os.Setenv("COLOR_ENABLED", "1")

		got := isColorEnabled()
		assert.True(t, got)
	})

	t.Run("COLOR_ENABLED == true", func(t *testing.T) {
		preRun()

		_ = os.Setenv("NO_COLOR", "")
		_ = os.Setenv("COLOR_ENABLED", "true")

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
			if tt.colorEnabled {
				t.Setenv("COLOR_ENABLED", "true")
			}

			if tt.is256color {
				t.Setenv("TERM", "256")
			}

			fn := makeColorFunc(tt.color)
			got := fn("text")

			require.Equal(t, tt.want, got)
		})
	}
}
