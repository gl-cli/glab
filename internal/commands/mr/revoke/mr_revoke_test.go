package revoke

import (
	"net/http"
	"testing"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/cli/internal/glinstance"
	"gitlab.com/gitlab-org/cli/internal/testing/cmdtest"
	"gitlab.com/gitlab-org/cli/internal/testing/httpmock"
	"gitlab.com/gitlab-org/cli/test"
)

func runCommand(t *testing.T, rt http.RoundTripper, cli string) (*test.CmdOut, error) {
	ios, _, stdout, stderr := cmdtest.TestIOStreams()

	factory := cmdtest.NewTestFactory(ios,
		cmdtest.WithGitLabClient(cmdtest.NewTestApiClient(t, &http.Client{Transport: rt}, "", glinstance.DefaultHostname).Lab()),
		cmdtest.WithBranch("current-branch"),
	)

	cmd := NewCmdRevoke(factory)

	return cmdtest.ExecuteCommand(cmd, cli, stdout, stderr)
}

func TestMrRevoke(t *testing.T) {
	type httpMock struct {
		method string
		path   string
		status int
		body   string
	}

	tests := []struct {
		name      string
		cli       string
		httpMocks []httpMock

		expectedOut string
	}{
		{
			name: "when an MR is unapproved using an MR id",
			cli:  "123",
			httpMocks: []httpMock{
				{
					http.MethodGet,
					"/api/v4/projects/OWNER/REPO/merge_requests/123",
					http.StatusOK,
					`{
								"id": 123,
								"iid": 123,
								"project_id": 3,
								"title": "test mr title",
								"description": "test mr description",
								"state": "opened"
							}`,
				},
				{
					http.MethodPost,
					"/api/v4/projects/OWNER/REPO/merge_requests/123/unapprove",
					http.StatusCreated,
					"{}",
				},
			},

			expectedOut: heredoc.Doc(`
				- Revoking approval for merge request !123...
				✓ Merge request approval revoked.
				`),
		},
		{
			name: "when an MR is unapproved using the current branch",
			cli:  "",
			httpMocks: []httpMock{
				{
					http.MethodGet,
					"/api/v4/projects/OWNER/REPO/merge_requests/123",
					http.StatusOK,
					`{
								"id": 123,
								"iid": 123,
								"project_id": 3,
								"title": "test mr title",
								"description": "test mr description",
								"state": "opened"
							}`,
				},
				{
					http.MethodGet,
					"/api/v4/projects/OWNER/REPO/merge_requests?per_page=30&source_branch=current-branch&state=opened",
					http.StatusOK,
					`[{
								"id": 123,
								"iid": 123,
								"project_id": 3,
								"title": "test mr title",
								"description": "test mr description",
								"state": "opened"
							}]`,
				},
				{
					http.MethodPost,
					"/api/v4/projects/OWNER/REPO/merge_requests/123/unapprove",
					http.StatusCreated,
					"{}",
				},
			},

			expectedOut: heredoc.Doc(`
				- Revoking approval for merge request !123...
				✓ Merge request approval revoked.
				`),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fakeHTTP := &httpmock.Mocker{
				MatchURL: httpmock.PathAndQuerystring,
			}
			defer fakeHTTP.Verify(t)

			for _, mock := range tc.httpMocks {
				fakeHTTP.RegisterResponder(mock.method, mock.path, httpmock.NewStringResponse(mock.status, mock.body))
			}

			output, err := runCommand(t, fakeHTTP, tc.cli)

			if assert.NoErrorf(t, err, "error running command `mr revoke %s`: %v", tc.cli, err) {
				out := output.String()

				assert.Equal(t, tc.expectedOut, out)
				assert.Empty(t, output.Stderr())
			}
		})
	}
}
