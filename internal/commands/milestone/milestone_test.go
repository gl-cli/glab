package milestone

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"gitlab.com/gitlab-org/cli/internal/testing/cmdtest"
)

func TestNewCmdMilestone(t *testing.T) {
	ios, _, _, _ := cmdtest.TestIOStreams()
	factory := cmdtest.NewTestFactory(ios)

	cmd := NewCmdMilestone(factory)

	assert.NotNil(t, cmd)
	assert.Equal(t, "milestone <command>", cmd.Use)
	assert.Equal(t, "Manage group or project milestones.", cmd.Short)
	assert.True(t, cmd.HasSubCommands())

	// Check that all expected subcommands are present
	subcommands := cmd.Commands()
	subcommandNames := make([]string, len(subcommands))
	for i, subcmd := range subcommands {
		subcommandNames[i] = subcmd.Use
	}

	expectedSubcommands := []string{"get", "list", "create", "edit", "delete"}
	for _, expected := range expectedSubcommands {
		assert.Contains(t, subcommandNames, expected)
	}
}
