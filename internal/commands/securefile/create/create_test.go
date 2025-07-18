package create

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

func Test_SecurefileCreate(t *testing.T) {
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
			Name:        "Create securefile",
			ExpectedMsg: []string{"• Creating secure file repo=OWNER/REPO fileName=newfile.txt", "✓ Secure file newfile.txt created."},
			cli:         "newfile.txt testdata/localfile.txt",
			httpMocks: []httpMock{
				{
					http.MethodPost,
					"/api/v4/projects/OWNER/REPO/secure_files",
					http.StatusOK,
					`{
						"id": 1,
						"name": "newfile.txt",
						"checksum": "16630b189ab34b2e3504f4758e1054d2e478deda510b2b08cc0ef38d12e80aac",
						"checksum_algorithm": "sha256",
						"created_at": "2022-02-22T22:22:22.000Z",
						"expires_at": null,
						"metadata": null
					}`,
				},
			},
		},
		{
			Name: "Create securefile but API errors",
			cli:  "newfile.txt testdata/localfile.txt",
			httpMocks: []httpMock{
				{
					http.MethodPost,
					"/api/v4/projects/OWNER/REPO/secure_files",
					http.StatusBadRequest,
					"",
				},
			},
			wantErr:    true,
			wantStderr: "Error creating secure file: POST https://gitlab.com/api/v4/projects/OWNER%2FREPO/secure_files: 400",
		},
		{
			Name:       "Get a securefile with invalid file path",
			cli:        "newfile.txt testdata/missingfile.txt",
			httpMocks:  []httpMock{},
			wantErr:    true,
			wantStderr: "Unable to read file at testdata/missingfile.txt: open testdata/missingfile.txt: no such file or directory",
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
		cmdtest.WithGitLabClient(cmdtest.NewTestApiClient(t, &http.Client{Transport: rt}, "", glinstance.DefaultHostname).Lab()),
	)
	cmd := NewCmdCreate(factory)
	return cmdtest.ExecuteCommand(cmd, cli, stdout, stderr)
}
