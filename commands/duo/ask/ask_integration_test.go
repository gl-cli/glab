package ask

import (
	"testing"

	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/pkg/prompt"
	"gitlab.com/gitlab-org/cli/test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cli/commands/cmdtest"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
)

func TestAskGit_Integration(t *testing.T) {
	glTestHost := test.GetHostOrSkip(t)
	t.Setenv("GITLAB_HOST", glTestHost)

	restore := prompt.StubConfirm(false)
	defer restore()

	cfg, err := config.Init()
	require.NoError(t, err)
	io, _, stdout, _ := cmdtest.TestIOStreams()
	f := cmdutils.NewFactory(io, false, cfg)

	cmd := NewCmdAsk(f)
	cli := "--git how to create a branch"
	_, err = cmdtest.RunCommand(cmd, cli)

	out := stdout.String()

	if err != nil {
		t.Fatalf("got unexpected error running 'glab duo ask %s': %s", cli, err)
	}

	for _, msg := range []string{"Commands", "Explanation", "git checkout"} {
		assert.Contains(t, out, msg)
	}
}
