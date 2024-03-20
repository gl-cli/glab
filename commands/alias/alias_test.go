package alias

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/test"
)

func Test_Alias(t *testing.T) {
	old := os.Stdout // keep backup of the real stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	assert.Nil(t, NewCmdAlias(&cmdutils.Factory{}).Execute())

	out := test.ReturnBuffer(old, r, w)

	assert.Contains(t, out, "Use \"alias [command] --help\" for more information about a command.\n")
}
