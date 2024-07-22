package ask

import (
	"testing"

	"gitlab.com/gitlab-org/cli/pkg/iostreams"
	"gitlab.com/gitlab-org/cli/pkg/prompt"
	"gitlab.com/gitlab-org/cli/test"

	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/cli/commands/cmdtest"
)

func TestAskGit_Integration(t *testing.T) {
	glTestHost := test.GetHostOrSkip(t)
	t.Setenv("GITLAB_HOST", glTestHost)

	restore := prompt.StubConfirm(false)
	defer restore()

	io, _, stdout, _ := iostreams.Test()

	f := cmdtest.StubFactory("")
	f.IO = io

	cmd := NewCmdAsk(f)
	cli := "--git how to create a branch"
	_, err := cmdtest.RunCommand(cmd, cli)

	out := stdout.String()

	if err != nil {
		t.Fatalf("got unexpected error running 'glab duo ask %s': %s", cli, err)
	}

	for _, msg := range []string{"Commands", "Explanation", "git checkout"} {
		assert.Contains(t, out, msg)
	}
}
