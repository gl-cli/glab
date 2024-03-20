package variable

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/test"
)

func TestNewVariableCmd(t *testing.T) {
	old := os.Stdout // keep backup of the real stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	assert.Nil(t, NewVariableCmd(&cmdutils.Factory{}).Execute())

	out := test.ReturnBuffer(old, r, w)

	assert.Contains(t, out, "Use \"variable [command] --help\" for more information about a command.\n")
}
