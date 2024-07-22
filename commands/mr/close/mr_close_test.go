package close

import (
	"io"
	"net/http"
	"testing"

	"github.com/google/shlex"

	"gitlab.com/gitlab-org/cli/pkg/iostreams"

	"github.com/MakeNowJust/heredoc/v2"

	"github.com/stretchr/testify/assert"
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

	cmd := NewCmdClose(factory)

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

func TestMrClose(t *testing.T) {
	fakeHTTP := httpmock.New()
	defer fakeHTTP.Verify(t)

	fakeHTTP.RegisterResponder(http.MethodGet, `/projects/OWNER/REPO/merge_requests/123`,
		httpmock.NewStringResponse(http.StatusOK, `{
			"id": 123,
			"iid": 123,
			"project_id": 3,
			"title": "test mr title",
			"description": "test mr description",
			"state": "opened"}`))

	fakeHTTP.RegisterResponder(http.MethodPut, `/projects/OWNER/REPO/merge_requests/123`,
		func(req *http.Request) (*http.Response, error) {
			rb, _ := io.ReadAll(req.Body)

			// ensure CLI updates MR to closed
			assert.Contains(t, string(rb), "\"state_event\":\"close\"")
			resp, _ := httpmock.NewStringResponse(http.StatusOK, `{
			"id": 123,
			"iid": 123,
			"project_id": 3,
			"title": "test mr title",
			"description": "test mr description",
			"state": "closed"}`)(req)
			return resp, nil
		},
	)

	mrID := "123"
	output, err := runCommand(fakeHTTP, mrID)
	if assert.NoErrorf(t, err, "error running command `mr close %s`", mrID) {
		out := output.String()

		assert.Equal(t, heredoc.Doc(`
		- Closing merge request...
		✓ Closed merge request !123.

		`), out)
		assert.Empty(t, output.Stderr())
	}
}
