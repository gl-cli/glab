package job

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cli/internal/testing/cmdtest"
)

func TestCIPipelineCancelWithoutArgument(t *testing.T) {
	t.Parallel()

	exec := cmdtest.SetupCmdForTest(t, NewCmdCancel)

	out, err := exec("")
	assert.EqualError(t, err, "You must pass a job ID.")

	assert.Empty(t, out.OutBuf.String())
	assert.Empty(t, out.ErrBuf.String())
}

func TestCIDryRunDeleteNothing(t *testing.T) {
	t.Parallel()

	exec := cmdtest.SetupCmdForTest(t, NewCmdCancel)

	out, err := exec("--dry-run 11111111,22222222")
	require.NoError(t, err)

	stdout := out.OutBuf.String()
	assert.Contains(t, stdout, "Job #11111111 will be canceled.")
	assert.Contains(t, stdout, "Job #22222222 will be canceled.")
	assert.Empty(t, out.ErrBuf.String())
}
