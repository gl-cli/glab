package config

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_GitDir(t *testing.T) {
	gotRelative := filepath.Join(GitDir(true)...)
	gotAbsolute := filepath.Join(GitDir(false)...)
	absRelative, err := filepath.Abs(gotRelative)
	assert.Equal(t, nil, err)
	assert.Equal(t, gotAbsolute, absRelative)
}

func Test_LocalConfigDir(t *testing.T) {
	got := LocalConfigDir()
	assert.ElementsMatch(t, []string{filepath.Join("..", "..", ".git"), "glab-cli"}, got)
}

func Test_LocalConfigFile(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		expectedPath := filepath.Join("..", "..", ".git", "glab-cli", "config.yml")
		got := LocalConfigFile()
		assert.Equal(t, expectedPath, got)
	})

	t.Run("modified-LocalConfigDir()", func(t *testing.T) {
		expectedPath := filepath.Join(".config", "glab-cli", "config.yml")

		LocalConfigDir = func() []string {
			return []string{".config", "glab-cli"}
		}

		got := LocalConfigFile()
		assert.Equal(t, expectedPath, got)
	})
}
