package remove

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cli/internal/testing/cmdtest"
	"gitlab.com/gitlab-org/cli/internal/testing/httpmock"
	"gitlab.com/gitlab-org/cli/test"
)

func Test_SecurefileRemove(t *testing.T) {
	type httpMock struct {
		method string
		path   string
		status int
		body   string
	}

	testCases := []struct {
		Name        string
		ExpectedMsg []string
		wantErr     bool
		cli         string
		wantStderr  string
		httpMocks   []httpMock
	}{
		{
			Name:        "Remove a secure file",
			ExpectedMsg: []string{"• Deleting secure file repo=OWNER/REPO fileID=1", "✓ Secure file 1 deleted."},
			cli:         "1 -y",
			httpMocks: []httpMock{
				{
					http.MethodDelete,
					"/api/v4/projects/OWNER/REPO/secure_files/1",
					http.StatusNoContent,
					"",
				},
			},
		},
		{
			Name: "Remove a secure file but API errors",
			cli:  "1 -y",
			httpMocks: []httpMock{
				{
					http.MethodDelete,
					"/api/v4/projects/OWNER/REPO/secure_files/1",
					http.StatusBadRequest,
					"",
				},
			},
			wantErr:    true,
			wantStderr: "Error removing secure file: DELETE https://gitlab.com/api/v4/projects/OWNER%2FREPO/secure_files/1: 400",
		},
		{
			Name:       "Remove a secure file with invalid file ID",
			cli:        "abc -y",
			httpMocks:  []httpMock{},
			wantErr:    true,
			wantStderr: "Secure file ID must be an integer: abc",
		},
		{
			Name:       "Remove a secure file without force delete when not running interactively",
			cli:        "1",
			httpMocks:  []httpMock{},
			wantErr:    true,
			wantStderr: "--yes or -y flag is required when not running interactively.",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			fakeHTTP := &httpmock.Mocker{
				MatchURL: httpmock.PathOnly,
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
		cmdtest.WithGitLabClient(cmdtest.NewTestApiClient(t, &http.Client{Transport: rt}, "", "gitlab.com").Lab()),
	)
	cmd := NewCmdRemove(factory)
	return cmdtest.ExecuteCommand(cmd, cli, stdout, stderr)
}
