package mr

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/cli/commands/cmdtest"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/test"
)

func TestMain(m *testing.M) {
	cmdtest.InitTest(m, "mr_cmd_test")
	cmdtest.InitTest(m, "mr_cmd_autofill")
}

func TestMrCmd_noARgs(t *testing.T) {
	old := os.Stdout // keep backup of the real stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	assert.Nil(t, NewCmdMR(&cmdutils.Factory{}).Execute())

	out := test.ReturnBuffer(old, r, w)

	assert.Contains(t, out, "Use \"mr [command] --help\" for more information about a command.\n")
}
