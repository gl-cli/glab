package version

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/cli/internal/api"
	"gitlab.com/gitlab-org/cli/internal/testing/cmdtest"
)

func Test_Version(t *testing.T) {
	ios, _, stdout, stderr := cmdtest.TestIOStreams()
	f := cmdtest.NewTestFactory(ios, cmdtest.WithBuildInfo(api.BuildInfo{Version: "v1.0.0", Commit: "abcdefgh"}))
	assert.Nil(t, NewCmdVersion(f).Execute())

	assert.Equal(t, "glab 1.0.0 (abcdefgh)\n", stdout.String())
	assert.Equal(t, "", stderr.String())
}
