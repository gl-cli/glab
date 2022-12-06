package delete

import (
	"net/http"
	"testing"

	"github.com/google/shlex"

	"gitlab.com/gitlab-org/cli/pkg/iostreams"

	"github.com/MakeNowJust/heredoc"

	"github.com/alecthomas/assert"
	"github.com/xanzy/go-gitlab"
	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/pkg/httpmock"
	"gitlab.com/gitlab-org/cli/test"
)

func runCommand(rt http.RoundTripper, cli string) (*test.CmdOut, error) {
	ios, _, stdout, stderr := iostreams.Test()

	factory := &cmdutils.Factory{
		IO: ios,
		HttpClient: func() (*gitlab.Client, error) {
			a, err := api.TestClient(&http.Client{Transport: rt}, "", "", false)
			if err != nil {
				return nil, err
			}
			return a.Lab(), err
		},
		BaseRepo: func() (glrepo.Interface, error) {
			return glrepo.New("OWNER", "REPO"), nil
		},
	}

	_, _ = factory.HttpClient()

	cmd := NewCmdDelete(factory)

	argv, err := shlex.Split(cli)
	if err != nil {
		return nil, err
	}
	cmd.SetArgs(argv)

	_, err = cmd.ExecuteC()
	return &test.CmdOut{
		OutBuf: stdout,
		ErrBuf: stderr,
	}, err
}

func TestCiDelete(t *testing.T) {
	fakeHTTP := httpmock.New()
	defer fakeHTTP.Verify(t)

	fakeHTTP.RegisterResponder("DELETE", "/api/v4/projects/OWNER/REPO/pipelines/11111111",
		httpmock.NewStringResponse(204, ""),
	)

	pipelineId := "11111111"
	output, err := runCommand(fakeHTTP, pipelineId)
	if err != nil {
		t.Errorf("error running command `ci delete %s`: %v", pipelineId, err)
	}

	out := output.String()

	assert.Equal(t, heredoc.Doc(`
		Deleting Pipeline #11111111
		✓ Pipeline #11111111 Deleted Successfully
		`), out)
	assert.Empty(t, output.Stderr())
}

func TestCiDeleteMultiple(t *testing.T) {
	fakeHTTP := httpmock.New()
	defer fakeHTTP.Verify(t)

	fakeHTTP.RegisterResponder("DELETE", "/api/v4/projects/OWNER/REPO/pipelines/11111111",
		httpmock.NewStringResponse(204, ""),
	)
	fakeHTTP.RegisterResponder("DELETE", "/api/v4/projects/OWNER/REPO/pipelines/22222222",
		httpmock.NewStringResponse(204, ""),
	)

	pipelineId := "11111111,22222222"
	output, err := runCommand(fakeHTTP, pipelineId)
	if err != nil {
		t.Errorf("error running command `ci delete %s`: %v", pipelineId, err)
	}

	out := output.String()

	assert.Equal(t, heredoc.Doc(`
		Deleting Pipeline #11111111
		✓ Pipeline #11111111 Deleted Successfully
		Deleting Pipeline #22222222
		✓ Pipeline #22222222 Deleted Successfully
		`), out)
	assert.Empty(t, output.Stderr())
}
