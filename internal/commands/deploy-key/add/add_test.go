package add

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cli/internal/glinstance"
	"gitlab.com/gitlab-org/cli/internal/testing/cmdtest"
	"gitlab.com/gitlab-org/cli/internal/testing/httpmock"
	"gitlab.com/gitlab-org/cli/test"
)

func Test_AddDeployKey(t *testing.T) {
	type httpMock struct {
		method string
		path   string
		status int
		body   string
	}

	apiEndpoint := "/api/v4/projects/OWNER%2FREPO/deploy_keys"

	testCases := []struct {
		Name        string
		ExpectedMsg []string
		wantErr     bool
		cli         string
		wantStderr  string
		httpMocks   []httpMock
	}{
		{
			Name:        "Add deploy key",
			ExpectedMsg: []string{"New deploy key added."},
			cli:         "testdata/testkey.key -t testkey",
			httpMocks: []httpMock{
				{
					http.MethodPost,
					apiEndpoint,
					http.StatusOK,
					`{
						"key": "ssh-rsa AAAA...",
						"id": 12,
						"title": "My deploy key",
						"can_push": true,
						"created_at": "2025-01-01T00:00:00.000Z",
						"expires_at": null
					}`,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			fakeHTTP := &httpmock.Mocker{
				MatchURL: httpmock.PathAndQuerystring,
			}
			defer fakeHTTP.Verify(t)

			for _, mock := range tc.httpMocks {
				fakeHTTP.RegisterResponder(mock.method, mock.path, httpmock.NewStringResponse(mock.status, mock.body))
			}

			out, err := runCommand(t, fakeHTTP, tc.cli)
			if tc.wantErr {
				if assert.Error(t, err) {
					require.Equal(t, tc.wantStderr, err.Error())
				}
				return
			}
			require.NoError(t, err)

			for _, msg := range tc.ExpectedMsg {
				require.Contains(t, out.Stderr(), msg)
			}
		})
	}
}

func runCommand(t *testing.T, rt http.RoundTripper, cli string) (*test.CmdOut, error) {
	ios, _, stdout, stderr := cmdtest.TestIOStreams()
	factory := cmdtest.NewTestFactory(ios,
		cmdtest.WithGitLabClient(cmdtest.NewTestApiClient(t, &http.Client{Transport: rt}, "", glinstance.DefaultHostname).Lab()),
	)
	cmd := NewCmdAdd(factory)
	return cmdtest.ExecuteCommand(cmd, cli, stdout, stderr)
}
