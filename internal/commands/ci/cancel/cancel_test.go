package cancel

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/cli/internal/testing/cmdtest"
	"gitlab.com/gitlab-org/cli/test"
)

func TestIssueCmd(t *testing.T) {
	old := os.Stdout // keep backup of the real stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	assert.Nil(t, NewCmdCancel(cmdtest.NewTestFactory(nil)).Execute())

	out := test.ReturnBuffer(old, r, w)

	assert.Contains(t, out, "Cancel CI/CD jobs.\n")
	assert.Contains(t, out, "Cancel CI/CD pipelines.\n")
}
