package securefile

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/cli/internal/testing/cmdtest"
	"gitlab.com/gitlab-org/cli/test"
)

func Test_Securefile(t *testing.T) {
	old := os.Stdout // keep backup of the real stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	defer func() { os.Stdout = old }()

	assert.Nil(t, NewCmdSecurefile(cmdtest.NewTestFactory(nil)).Execute())

	out := test.ReturnBuffer(old, r, w)

	assert.Contains(t, out, "Use \"securefile [command] --help\" for more information about a command.\n")
}
