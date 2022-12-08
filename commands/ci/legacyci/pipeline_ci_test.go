// This package contains the old `glab pipeline ci` command which has been deprecated
// in favour of the `glab ci` command.
// This package is kept for backward compatibility but issues a deprecation warning
package legacyci

import (
	"testing"

	"gitlab.com/gitlab-org/cli/pkg/iostreams"

	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
)

func TestNewCmdCI(t *testing.T) {
	ioStrm, stdin, stdout, stderr := iostreams.Test()

	cmd := NewCmdCI(&cmdutils.Factory{
		IO: ioStrm,
	})

	cmd.SetIn(stdin)
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)

	_, err := cmd.ExecuteC()

	assert.Nil(t, err)

	assert.Contains(t, stdout.String(), "Work with GitLab CI/CD pipelines and jobs\n")
	assert.Contains(t, stderr.String(), "")
	assert.Contains(t, stdout.String(), "This command is deprecated. All the commands under it has been moved to `ci` or `pipeline` command. See https://gitlab.com/gitlab-org/cli/issues/372 for more info.\n")
}
