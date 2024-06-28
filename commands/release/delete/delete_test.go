package delete

import (
	"net/http"
	"testing"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/cli/pkg/httpmock"

	"gitlab.com/gitlab-org/cli/commands/cmdtest"

	"gitlab.com/gitlab-org/cli/test"
)

func runCommand(rt http.RoundTripper, isTTY bool, cli string) (*test.CmdOut, error) {
	ios, _, stdout, stderr := cmdtest.InitIOStreams(isTTY, "")

	factory := cmdtest.InitFactory(ios, rt)

	_, _ = factory.HttpClient()

	cmd := NewCmdDelete(factory)

	return cmdtest.ExecuteCommand(cmd, cli, stdout, stderr)
}

func TestReleaseDelete(t *testing.T) {
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
			name: "delete a release",
			cli:  "0.0.1 --yes",
			httpMocks: []httpMock{
				{
					http.MethodGet,
					"/api/v4/projects/OWNER/REPO/releases/0.0.1",
					http.StatusOK,
					`{
							  "name": "test_release",
							  "tag_name": "0.0.1",
							  "description": null,
							  "created_at": "2020-01-23T07:13:17.721Z",
							  "released_at": "2020-01-23T07:13:17.721Z",
							  "upcoming_release": false
							}`,
				},
				{
					http.MethodDelete,
					"/api/v4/projects/OWNER/REPO/releases/0.0.1",
					http.StatusOK,
					`{
							  "name": "test_release",
							  "tag_name": "0.0.1",
							  "description": null,
							  "created_at": "2020-01-23T07:13:17.721Z",
							  "released_at": "2020-01-23T07:13:17.721Z",
							  "upcoming_release": false
							}`,
				},
			},

			expectedOut: heredoc.Doc(`• Validating tag repo=OWNER/REPO tag=0.0.1
												• Deleting release repo=OWNER/REPO tag=0.0.1
												✓ Release "test_release" deleted.
											`),
		},
		{
			name: "delete a release and associated tag",
			cli:  "0.0.1 --yes --with-tag",
			httpMocks: []httpMock{
				{
					http.MethodGet,
					"/api/v4/projects/OWNER/REPO/releases/0.0.1",
					http.StatusOK,
					`{
							  "name": "test_release",
							  "tag_name": "0.0.1",
							  "description": null,
							  "created_at": "2020-01-23T07:13:17.721Z",
							  "released_at": "2020-01-23T07:13:17.721Z",
							  "upcoming_release": false
							}`,
				},
				{
					http.MethodDelete,
					"/api/v4/projects/OWNER/REPO/releases/0.0.1",
					http.StatusOK,
					`{
							  "name": "test_release",
							  "tag_name": "0.0.1",
							  "description": null,
							  "created_at": "2020-01-23T07:13:17.721Z",
							  "released_at": "2020-01-23T07:13:17.721Z",
							  "upcoming_release": false
							}`,
				},
				{
					http.MethodDelete,
					"/api/v4/projects/OWNER/REPO/repository/tags/0.0.1",
					http.StatusOK,
					`{
							  "name": "test_release",
							  "tag_name": "0.0.1",
							  "description": null,
							  "created_at": "2020-01-23T07:13:17.721Z",
							  "released_at": "2020-01-23T07:13:17.721Z",
							  "upcoming_release": false
							}`,
				},
			},

			expectedOut: heredoc.Doc(`• Validating tag repo=OWNER/REPO tag=0.0.1
												• Deleting release repo=OWNER/REPO tag=0.0.1
												✓ Release "test_release" deleted.
												• Deleting associated tag "0.0.1".
												✓ Tag "test_release" deleted.
											`),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fakeHTTP := httpmock.New()
			defer fakeHTTP.Verify(t)

			for _, mock := range tc.httpMocks {
				fakeHTTP.RegisterResponder(mock.method, mock.path, httpmock.NewStringResponse(mock.status, mock.body))
			}

			output, err := runCommand(fakeHTTP, false, tc.cli)

			if assert.NoErrorf(t, err, "error running command `delete %s`: %v", tc.cli, err) {
				assert.Equal(t, tc.expectedOut, output.Stderr())
				assert.Empty(t, output.String())
			}
		})
	}
}
