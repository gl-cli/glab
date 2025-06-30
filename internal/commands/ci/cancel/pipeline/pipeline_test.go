package pipeline

import (
	"net/http"
	"testing"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/cli/internal/commands/cmdtest"
	"gitlab.com/gitlab-org/cli/internal/httpmock"
	"gitlab.com/gitlab-org/cli/test"
)

func runCommand(rt http.RoundTripper, args string) (*test.CmdOut, error) {
	ios, _, stdout, stderr := cmdtest.TestIOStreams()
	factory := cmdtest.InitFactory(ios, rt)

	cmd := NewCmdCancel(factory)

	return cmdtest.ExecuteCommand(cmd, args, stdout, stderr)
}

func TestCIPipelineCancelWithoutArgument(t *testing.T) {
	fakeHTTP := httpmock.New()
	fakeHTTP.MatchURL = httpmock.PathAndQuerystring
	defer fakeHTTP.Verify(t)

	pipelineId := ""
	output, err := runCommand(fakeHTTP, pipelineId)
	assert.EqualError(t, err, "You must pass a pipeline ID.")

	assert.Empty(t, output.String())
	assert.Empty(t, output.Stderr())
}

func TestCIDryRunDeleteNothing(t *testing.T) {
	fakeHTTP := httpmock.New()
	defer fakeHTTP.Verify(t)

	args := "--dry-run 11111111,22222222"
	output, err := runCommand(fakeHTTP, args)
	if err != nil {
		t.Errorf("error running command `ci cancel pipeline %s`: %v", args, err)
	}

	out := output.String()

	assert.Contains(t, heredoc.Doc(`
	• Pipeline #11111111 will be canceled.
	• Pipeline #22222222 will be canceled.
	`), out)
	assert.Empty(t, output.Stderr())
}
