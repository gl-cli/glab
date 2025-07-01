package board

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/cli/internal/testing/cmdtest"
	"gitlab.com/gitlab-org/cli/test"
)

func TestNewCmdBoard(t *testing.T) {
	old := os.Stdout // keep backup of the real stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	assert.Nil(t, NewCmdBoard(cmdtest.NewTestFactory(nil)).Execute())

	out := test.ReturnBuffer(old, r, w)

	assert.Contains(t, out, "Work with GitLab Issue Boards in the given project.\n")
}
