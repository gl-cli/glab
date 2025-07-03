package reopen

import (
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/google/shlex"
	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/cli/internal/commands/issuable"
	"gitlab.com/gitlab-org/cli/internal/testing/cmdtest"
	"gitlab.com/gitlab-org/cli/internal/testing/httpmock"
	"gitlab.com/gitlab-org/cli/test"
)

func mockAllResponses(t *testing.T, fakeHTTP *httpmock.Mocker) {
	fakeHTTP.RegisterResponder(http.MethodGet, "/projects/OWNER/REPO/issues/1",
		httpmock.NewStringResponse(http.StatusOK, `{
			"id": 1,
			"iid": 1,
			"title": "test issue",
			"state": "closed",
			"issue_type": "issue",
			"created_at": "2023-04-05T10:51:26.371Z"
		}`),
	)

	fakeHTTP.RegisterResponder(http.MethodPut, "/projects/OWNER/REPO/issues/1",
		func(req *http.Request) (*http.Response, error) {
			rb, _ := io.ReadAll(req.Body)

			assert.Contains(t, string(rb), `"state_event":"reopen"`)
			resp, _ := httpmock.NewStringResponse(http.StatusOK, `{
				"id": 1,
				"iid": 1,
				"state": "open",
				"issue_type": "issue",
				"created_at": "2023-04-05T10:51:26.371Z"
			}`)(req)

			return resp, nil
		},
	)

	fakeHTTP.RegisterResponder(http.MethodGet, "/projects/OWNER/REPO/issues/2",
		httpmock.NewStringResponse(http.StatusOK, `{
			"id": 2,
			"iid": 2,
			"title": "test incident",
			"state": "closed",
			"issue_type": "incident",
			"created_at": "2023-04-05T10:51:26.371Z"
		}`),
	)

	fakeHTTP.RegisterResponder(http.MethodPut, "/projects/OWNER/REPO/issues/2",
		func(req *http.Request) (*http.Response, error) {
			rb, _ := io.ReadAll(req.Body)

			assert.Contains(t, string(rb), `"state_event":"reopen"`)
			resp, _ := httpmock.NewStringResponse(http.StatusOK, `{
				"id": 2,
				"iid": 2,
				"state": "opened",
				"issue_type": "incident",
				"created_at": "2023-04-05T10:51:26.371Z"
			}`)(req)

			return resp, nil
		},
	)

	fakeHTTP.RegisterResponder(http.MethodGet, "/projects/OWNER/REPO/issues/404",
		httpmock.NewStringResponse(http.StatusNotFound, `{"message": "404 not found"}`),
	)
}

func runCommand(t *testing.T, rt http.RoundTripper, issuableID string, issueType issuable.IssueType) (*test.CmdOut, error) {
	ios, _, stdout, stderr := cmdtest.TestIOStreams()

	factory := cmdtest.NewTestFactory(ios,
		cmdtest.WithGitLabClient(cmdtest.NewTestApiClient(t, &http.Client{Transport: rt}, "", "").Lab()),
		cmdtest.WithBaseRepo("OWNER", "REPO"),
	)

	cmd := NewCmdReopen(factory, issueType)

	argv, err := shlex.Split(issuableID)
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

func TestIssuableReopen(t *testing.T) {
	tests := []struct {
		iid        int
		name       string
		issueType  issuable.IssueType
		wantOutput string
		wantErr    bool
	}{
		{
			iid:       1,
			name:      "issue_reopen",
			issueType: issuable.TypeIssue,
			wantOutput: heredoc.Doc(`
				- Reopening issue...
				✓ Reopened issue #1.

				`),
		},
		{
			iid:       2,
			name:      "incident_reopen",
			issueType: issuable.TypeIncident,
			wantOutput: heredoc.Doc(`
				- Reopening incident...
				✓ Reopened incident #2.

				`),
		},
		{
			iid:       2,
			name:      "incident_reopen_using_issue_command",
			issueType: issuable.TypeIssue,
			wantOutput: heredoc.Doc(`
				- Reopening issue...
				✓ Reopened issue #2.

				`),
		},
		{
			iid:        1,
			name:       "issue_reopen_using_incident_command",
			issueType:  issuable.TypeIncident,
			wantOutput: "Incident not found, but an issue with the provided ID exists. Run `glab issue reopen <id>` to reopen.\n",
		},
		{
			iid:        404,
			name:       "issue_not_found",
			issueType:  issuable.TypeIssue,
			wantOutput: "404 Not Found",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		fakeHTTP := httpmock.New()

		mockAllResponses(t, fakeHTTP)

		t.Run(tt.name, func(t *testing.T) {
			output, err := runCommand(t, fakeHTTP, fmt.Sprint(tt.iid), tt.issueType)
			if tt.wantErr {
				assert.Contains(t, err.Error(), tt.wantOutput)
				return
			}

			assert.NoErrorf(t, err, "error running command `%s reopen %d`.", tt.issueType, tt.iid)
			assert.Equal(t, tt.wantOutput, output.String())
			assert.Empty(t, output.Stderr())
		})
	}
}
