package delete

import (
	"net/http"
	"testing"

	"gitlab.com/gitlab-org/cli/pkg/iostreams"

	"github.com/MakeNowJust/heredoc"

	"github.com/alecthomas/assert"
	"gitlab.com/gitlab-org/cli/commands/cmdtest"
	"gitlab.com/gitlab-org/cli/pkg/httpmock"
	"gitlab.com/gitlab-org/cli/test"
)

func runCommand(rt http.RoundTripper, cli string) (*test.CmdOut, error) {
	ios, _, stdout, stderr := iostreams.Test()
	factory := cmdtest.InitFactory(ios, rt)

	_, _ = factory.HttpClient()

	cmd := NewCmdDelete(factory)

	return cmdtest.ExecuteCommand(cmd, cli, stdout, stderr)
}

func TestCiDelete(t *testing.T) {
	fakeHTTP := httpmock.New()
	defer fakeHTTP.Verify(t)

	fakeHTTP.RegisterResponder(http.MethodDelete, "/api/v4/projects/OWNER/REPO/pipelines/11111111",
		httpmock.NewStringResponse(http.StatusNoContent, ""),
	)

	pipelineId := "11111111"
	output, err := runCommand(fakeHTTP, pipelineId)
	if err != nil {
		t.Errorf("error running command `ci delete %s`: %v", pipelineId, err)
	}

	out := output.String()

	assert.Equal(t, heredoc.Doc(`
		Deleting pipeline #11111111
		✓ Pipeline #11111111 deleted successfully
		`), out)
	assert.Empty(t, output.Stderr())
}

func TestCiDeleteMultiple(t *testing.T) {
	fakeHTTP := httpmock.New()
	defer fakeHTTP.Verify(t)

	fakeHTTP.RegisterResponder(http.MethodDelete, "/api/v4/projects/OWNER/REPO/pipelines/11111111",
		httpmock.NewStringResponse(http.StatusNoContent, ""),
	)
	fakeHTTP.RegisterResponder(http.MethodDelete, "/api/v4/projects/OWNER/REPO/pipelines/22222222",
		httpmock.NewStringResponse(http.StatusNoContent, ""),
	)

	pipelineId := "11111111,22222222"
	output, err := runCommand(fakeHTTP, pipelineId)
	if err != nil {
		t.Errorf("error running command `ci delete %s`: %v", pipelineId, err)
	}

	out := output.String()

	assert.Equal(t, heredoc.Doc(`
		Deleting pipeline #11111111
		✓ Pipeline #11111111 deleted successfully
		Deleting pipeline #22222222
		✓ Pipeline #22222222 deleted successfully
		`), out)
	assert.Empty(t, output.Stderr())
}
