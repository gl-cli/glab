package iteration

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/cli/internal/testing/cmdtest"
	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/test"
)

func TestNewCmdIteration(t *testing.T) {
	old := os.Stdout // keep backup of the real stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	assert.Nil(t, NewCmdIteration(&cmdtest.Factory{ConfigStub: func() config.Config { return config.NewBlankConfig() }}).Execute())

	out := test.ReturnBuffer(old, r, w)

	assert.Contains(t, out, "Use \"iteration [command] --help\" for more information about a command.\n")
}
