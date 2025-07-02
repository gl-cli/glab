package delete

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

func Test_DeployKeyRemove(t *testing.T) {
	type httpMock struct {
		method string
		path   string
		status int
		body   string
	}

	apiEndpoint := "/api/v4/projects/OWNER%2FREPO/deploy_keys/123"

	testCases := []struct {
		Name        string
		ExpectedMsg []string
		wantErr     bool
		cli         string
		wantStderr  string
		httpMocks   []httpMock
	}{
		{
			Name:        "Remove a deploy key",
			ExpectedMsg: []string{"Deploy key deleted.\n"},
			cli:         "123",
			httpMocks: []httpMock{
				{
					http.MethodDelete,
					apiEndpoint,
					http.StatusNoContent,
					"",
				},
			},
		},
		{
			Name:       "Remove a deploy key with invalid file ID",
			cli:        "abc",
			httpMocks:  []httpMock{},
			wantErr:    true,
			wantStderr: "Deploy key ID must be an integer: abc",
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
	cmd := NewCmdDelete(factory)
	return cmdtest.ExecuteCommand(cmd, cli, stdout, stderr)
}
