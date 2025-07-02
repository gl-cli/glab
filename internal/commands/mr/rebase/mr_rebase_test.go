package rebase

import (
	"net/http"
	"testing"

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

	cmd := NewCmdRebase(factory)

	return cmdtest.ExecuteCommand(cmd, cli, stdout, stderr)
}

func TestMrRebase(t *testing.T) {
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
			name: "when an MR is rebased using an MR id",
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
					http.MethodPut,
					"/api/v4/projects/OWNER/REPO/merge_requests/123/rebase",
					http.StatusAccepted,
					`{ "rebase_in_progress": true }`,
				},
				{
					http.MethodGet,
					"/api/v4/projects/OWNER/REPO/merge_requests/123?include_rebase_in_progress=true",
					http.StatusOK,
					`{
							 "rebase_in_progress": false,
							 "merge_error": null
						   }`,
				},
			},

			expectedOut: "✓ Rebase successful!\n",
		},
		{
			name: "when an MR is rebased with skip-ci flag",
			cli:  "123 --skip-ci",
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
					http.MethodPut,
					"/api/v4/projects/OWNER/REPO/merge_requests/123/rebase",
					http.StatusAccepted,
					`{ "rebase_in_progress": true }`,
				},
				{
					http.MethodGet,
					"/api/v4/projects/OWNER/REPO/merge_requests/123?include_rebase_in_progress=true",
					http.StatusOK,
					`{
							 "rebase_in_progress": false,
							 "merge_error": null
						   }`,
				},
			},

			expectedOut: "✓ Rebase successful!\n",
		},
		{
			name: "when an MR is rebased using current branch",
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
					http.MethodPut,
					"/api/v4/projects/OWNER/REPO/merge_requests/123/rebase",
					http.StatusAccepted,
					`{ "rebase_in_progress": true }`,
				},
				{
					http.MethodGet,
					"/api/v4/projects/OWNER/REPO/merge_requests/123?include_rebase_in_progress=true",
					http.StatusOK,
					`{
							 "rebase_in_progress": false,
							 "merge_error": null
						   }`,
				},
			},

			expectedOut: "✓ Rebase successful!\n",
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

			if assert.NoErrorf(t, err, "error running command `mr rebase %s`: %v", tc.cli, err) {
				out := output.String()

				assert.Equal(t, tc.expectedOut, out)
				assert.Empty(t, output.Stderr())
			}
		})
	}
}
