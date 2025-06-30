package approvers

import (
	"net/http"
	"testing"

	"github.com/google/shlex"
	"gitlab.com/gitlab-org/cli/internal/testing/cmdtest"

	"github.com/MakeNowJust/heredoc/v2"

	"github.com/stretchr/testify/assert"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/internal/glinstance"
	"gitlab.com/gitlab-org/cli/internal/httpmock"
	"gitlab.com/gitlab-org/cli/test"
)

func runCommand(rt http.RoundTripper, cli string) (*test.CmdOut, error) {
	ios, _, stdout, stderr := cmdtest.TestIOStreams()

	factory := &cmdtest.Factory{
		IOStub: ios,
		HttpClientStub: func() (*gitlab.Client, error) {
			a, err := cmdtest.TestClient(&http.Client{Transport: rt}, "", "", false)
			if err != nil {
				return nil, err
			}
			return a.Lab(), err
		},
		BaseRepoStub: func() (glrepo.Interface, error) {
			return glrepo.New("OWNER", "REPO", glinstance.DefaultHostname), nil
		},
	}

	cmd := NewCmdApprovers(factory)

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

func TestMrApprove(t *testing.T) {
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

	fakeHTTP.RegisterResponder(http.MethodGet, `/projects/OWNER/REPO/merge_requests/123/approval_state`,
		httpmock.NewFileResponse(http.StatusOK, "./testdata/approvalState.json"))

	mrID := "123"
	output, err := runCommand(fakeHTTP, mrID)
	if assert.NoErrorf(t, err, "error running command `mr approvers %s`", mrID) {
		out := output.String()

		assert.Equal(t, heredoc.Doc(`

		Listing merge request !123 eligible approvers:
		Approval rules overwritten.
		Rule "All Members" sufficient approvals (1/1 required):
		Name	Username	Approved
		Abc Approver	approver_1	-	
		Bar Approver	approver_2	-	
		Foo Reviewer	foo_reviewer	👍	

		`), out)
		assert.Empty(t, output.Stderr())
	}
}
