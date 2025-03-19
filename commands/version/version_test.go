package version

import (
	"testing"

	"gitlab.com/gitlab-org/cli/pkg/iostreams"

	"github.com/stretchr/testify/assert"
)

func Test_Version(t *testing.T) {
	ios, _, stdout, stderr := iostreams.Test()
	assert.Nil(t, NewCmdVersion(ios, "v1.0.0", "abcdefgh").Execute())

	assert.Equal(t, "glab 1.0.0 (abcdefgh)\n", stdout.String())
	assert.Equal(t, "", stderr.String())
}
