package ci

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/test"
)

func TestPipelineCmd(t *testing.T) {
	old := os.Stdout // keep backup of the real stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	assert.Nil(t, NewCmdCI(&cmdutils.Factory{}).Execute())

	out := test.ReturnBuffer(old, r, w)

	assert.Contains(t, out, "Use \"ci [command] --help\" for more information about a command.\n")
}
