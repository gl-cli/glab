//go:build !integration

package members

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/cli/internal/testing/cmdtest"
)

func TestNewCmdMembers(t *testing.T) {
	ios, _, _, _ := cmdtest.TestIOStreams()
	factory := cmdtest.NewTestFactory(ios)

	cmd := NewCmdMembers(factory)

	assert.Equal(t, "members <command> [flags]", cmd.Use)
	assert.Equal(t, "Manage project members.", cmd.Short)
	assert.Equal(t, "Add or remove members from a GitLab project.\n", cmd.Long)

	// Check that subcommands are added
	subcommands := cmd.Commands()
	assert.Len(t, subcommands, 2)

	var addCmd, removeCmd bool
	for _, subcmd := range subcommands {
		switch subcmd.Use {
		case "add [flags]":
			addCmd = true
		case "remove [flags]":
			removeCmd = true
		}
	}

	assert.True(t, addCmd, "add subcommand should be present")
	assert.True(t, removeCmd, "remove subcommand should be present")
}
