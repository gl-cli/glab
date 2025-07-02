package download

import (
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cli/internal/testing/cmdtest"
	"gitlab.com/gitlab-org/cli/internal/testing/httpmock"
	"gitlab.com/gitlab-org/cli/test"
)

func Test_SecurefileDownload(t *testing.T) {
	type httpMock struct {
		method string
		path   string
		status int
	}

	testCases := []struct {
		Name                 string
		ExpectedMsg          []string
		ExpectedFileLocation string
		wantErr              bool
		cli                  string
		wantStderr           string
		httpMocks            []httpMock
	}{
		{
			Name:                 "Download secure file to current folder",
			ExpectedMsg:          []string{"Downloaded secure file with ID 1"},
			ExpectedFileLocation: "downloaded.tmp",
			cli:                  "1",
			httpMocks: []httpMock{
				{
					http.MethodGet,
					"/api/v4/projects/OWNER/REPO/secure_files/1/download",
					http.StatusOK,
				},
			},
		},
		{
			Name:                 "Download secure file to custom folder",
			ExpectedMsg:          []string{"Downloaded secure file with ID 1"},
			ExpectedFileLocation: "newdir/new.txt",
			cli:                  "1 --path=newdir/new.txt",
			httpMocks: []httpMock{
				{
					http.MethodGet,
					"/api/v4/projects/OWNER/REPO/secure_files/1/download",
					http.StatusOK,
				},
			},
		},
	}

	defer os.Remove("downloaded.tmp")
	defer os.Remove("newdir/new.txt")

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			fakeHTTP := &httpmock.Mocker{
				MatchURL: httpmock.PathOnly,
			}
			defer fakeHTTP.Verify(t)

			for _, mock := range tc.httpMocks {
				fakeHTTP.RegisterResponder(mock.method, mock.path, httpmock.NewFileResponse(mock.status, "testdata/localfile.txt"))
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
				require.Contains(t, out.String(), msg)
			}

			_, err = os.Stat(tc.ExpectedFileLocation)
			require.NoError(t, err)

			actualContent, err := os.ReadFile(tc.ExpectedFileLocation)
			require.NoError(t, err, "Failed to read downloaded test file")

			assert.Equal(t, "Hello", string(actualContent), "File content should match")
		})
	}
}

func runCommand(t *testing.T, rt http.RoundTripper, cli string) (*test.CmdOut, error) {
	ios, _, stdout, stderr := cmdtest.TestIOStreams()
	factory := cmdtest.NewTestFactory(ios,
		cmdtest.WithGitLabClient(cmdtest.NewTestApiClient(t, &http.Client{Transport: rt}, "", "gitlab.com").Lab()),
	)
	cmd := NewCmdDownload(factory)
	return cmdtest.ExecuteCommand(cmd, cli, stdout, stderr)
}
