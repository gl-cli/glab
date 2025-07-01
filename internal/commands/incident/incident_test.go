package incident

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/cli/internal/testing/cmdtest"
	"gitlab.com/gitlab-org/cli/test"
)

func TestIncidentCmd(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	assert.Nil(t, NewCmdIncident(cmdtest.NewTestFactory(nil)).Execute())

	out := test.ReturnBuffer(old, r, w)

	assert.Contains(t, out, "Work with GitLab incidents.\n")
}
