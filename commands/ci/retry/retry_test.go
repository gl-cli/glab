package retry

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

	cmd := NewCmdRetry(factory)

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

func TestCiRetry(t *testing.T) {
	fakeHTTP := httpmock.New()
	defer fakeHTTP.Verify(t)

	// test will fail with unmatched HTTP stub if this POST is not performed
	fakeHTTP.RegisterResponder("POST", "/projects/OWNER/REPO/jobs/1122/retry",
		httpmock.NewStringResponse(201, `
		{
			"id": 1123,
			"status": "pending",
			"stage": "build",
			"name": "build-job",
			"ref": "branch-name",
			"tag": false,
			"coverage": null,
			"allow_failure": false,
			"created_at": "2022-12-01T05:13:13.703Z",
			"web_url": "https://gitlab.com/OWNER/REPO/-/jobs/1123"
		}
	`))

	jobId := "1122"
	output, err := runCommand(fakeHTTP, jobId)
	if err != nil {
		t.Errorf("error running command `ci retry %s`: %v", jobId, err)
	}

	out := output.String()

	assert.Equal(t, heredoc.Doc(`
		Retried job (id: 1123 ), status: pending , ref: branch-name , weburl:  https://gitlab.com/OWNER/REPO/-/jobs/1123 )
`), out)
	assert.Empty(t, output.Stderr())
}
