package version

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/cli/commands/cmdtest"
)

func Test_Version(t *testing.T) {
	ios, _, stdout, stderr := cmdtest.TestIOStreams()
	assert.Nil(t, NewCmdVersion(ios, "v1.0.0", "abcdefgh").Execute())

	assert.Equal(t, "glab 1.0.0 (abcdefgh)\n", stdout.String())
	assert.Equal(t, "", stderr.String())
}
