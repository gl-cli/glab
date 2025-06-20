package release

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/cli/commands/cmdtest"
	"gitlab.com/gitlab-org/cli/test"
)

func Test_Release(t *testing.T) {
	old := os.Stdout // keep backup of the real stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd := NewCmdRelease(&cmdtest.Factory{})
	assert.NotNil(t, cmd.Root())
	assert.Nil(t, cmd.Execute())

	out := test.ReturnBuffer(old, r, w)

	assert.Contains(t, out, "Manage GitLab releases")
}
