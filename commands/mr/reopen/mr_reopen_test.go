package reopen

import (
	"io"
	"net/http"
	"testing"

	"gitlab.com/gitlab-org/cli/commands/cmdtest"

	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/cli/pkg/httpmock"
	"gitlab.com/gitlab-org/cli/test"
)

func runCommand(rt http.RoundTripper, isTTY bool, cli string) (*test.CmdOut, error) {
	ios, _, stdout, stderr := cmdtest.InitIOStreams(isTTY, "")

	factory := cmdtest.InitFactory(ios, rt)

	_, _ = factory.HttpClient()

	cmd := NewCmdReopen(factory)

	return cmdtest.ExecuteCommand(cmd, cli, stdout, stderr)
}

func TestMrReopen(t *testing.T) {
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

		expectedPUTBody string
		expectedOut     string
	}{
		{
			name: "when an MR is reopened using an MR id",
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
								"state": "closed"
							}`,
				},
			},

			expectedPUTBody: `"state_event":"reopen"`,
			expectedOut:     "- Reopening merge request !123...\n✓ Reopened merge request !123.\n\n",
		},
		{
			name: "when an MR is reopened using a branch name",
			cli:  "foo",
			httpMocks: []httpMock{
				{
					http.MethodGet,
					"/api/v4/projects/OWNER/REPO/merge_requests?per_page=30&source_branch=foo&state=closed",
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
					http.MethodGet,
					"/api/v4/projects/OWNER/REPO/merge_requests/123",
					http.StatusOK,
					`{
								"id": 123,
								"iid": 123,
								"project_id": 3,
								"title": "test mr title",
								"description": "test mr description",
								"state": "closed"
							}`,
				},
			},

			expectedPUTBody: `"state_event":"reopen"`,
			expectedOut:     "- Reopening merge request !123...\n✓ Reopened merge request !123.\n\n",
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

			fakeHTTP.RegisterResponder(http.MethodPut, "/api/v4/projects/OWNER/REPO/merge_requests/123",
				func(req *http.Request) (*http.Response, error) {
					rb, _ := io.ReadAll(req.Body)

					// ensure CLI updates MR to reopen
					assert.Contains(t, string(rb), tc.expectedPUTBody)
					resp, _ := httpmock.NewStringResponse(http.StatusOK, `{
						"id": 123,
						"iid": 123,
						"project_id": 3,
						"title": "test mr title",
						"description": "test mr description",
						"state": "opened"}`)(req)
					return resp, nil
				},
			)

			output, err := runCommand(fakeHTTP, false, tc.cli)

			if assert.NoErrorf(t, err, "error running command `mr reopen %s`: %v", tc.cli, err) {
				out := output.String()

				assert.Equal(t, tc.expectedOut, out)
				assert.Empty(t, output.Stderr())
			}
		})
	}
}
