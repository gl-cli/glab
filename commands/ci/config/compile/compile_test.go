package compile

import (
	"fmt"
	"net/http"
	"path"
	"testing"

	"gitlab.com/gitlab-org/cli/commands/cmdtest"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/pkg/httpmock"
	"gitlab.com/gitlab-org/cli/test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_compileRun(t *testing.T) {
	type httpMock struct {
		method string
		path   string
		status int
		body   string
	}

	tests := []struct {
		name             string
		testFile         string
		StdOut           string
		wantErr          bool
		errMsg           string
		httpMocks        []httpMock
		showHaveBaseRepo bool
	}{
		{
			name:             "with invalid path specified",
			testFile:         "WRONG_PATH",
			StdOut:           "",
			wantErr:          true,
			errMsg:           "WRONG_PATH: no such file or directory",
			showHaveBaseRepo: true,
			httpMocks: []httpMock{
				{
					http.MethodGet,
					"/api/v4/projects/OWNER/REPO",
					http.StatusOK,
					`{
						"id": 123,
						"iid": 123
					}`,
				},
			},
		},
		{
			name:             "without base repo",
			testFile:         ".gitlab.ci.yml",
			StdOut:           "",
			wantErr:          true,
			errMsg:           "You must be in a GitLab project repository for this action: no base repository present",
			showHaveBaseRepo: false,
			httpMocks:        []httpMock{},
		},
		{
			name:             "when a valid path is specified and yaml is valid",
			testFile:         ".gitlab-ci.yml",
			StdOut:           "",
			wantErr:          false,
			errMsg:           "",
			showHaveBaseRepo: true,
			httpMocks: []httpMock{
				{
					http.MethodGet,
					"/api/v4/projects/OWNER/REPO",
					http.StatusOK,
					`{
						"id": 123,
						"iid": 123
					}`,
				},
				{
					http.MethodPost,
					"/api/v4/projects/123/ci/lint",
					http.StatusOK,
					`{
						"valid": true
					}`,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeHTTP := httpmock.New()
			defer fakeHTTP.Verify(t)

			for _, mock := range tt.httpMocks {
				fakeHTTP.RegisterResponder(mock.method, mock.path, httpmock.NewStringResponse(mock.status, mock.body))
			}

			args := path.Join(cmdtest.ProjectPath, "test/testdata", tt.testFile)

			result, err := runCommand(t, fakeHTTP, false, args, tt.showHaveBaseRepo)
			if tt.wantErr {
				require.Contains(t, err.Error(), tt.errMsg)
				return
			}
			require.NoError(t, err)

			assert.Equal(t, tt.StdOut, result.String())
		})
	}
}

func runCommand(t *testing.T, rt http.RoundTripper, isTTY bool, cli string, showHaveBaseRepo bool) (*test.CmdOut, error) {
	ios, _, stdout, stderr := cmdtest.InitIOStreams(isTTY, "")

	factory := cmdtest.InitFactory(ios, rt)

	if !showHaveBaseRepo {
		factory.BaseRepo = func() (glrepo.Interface, error) {
			return nil, fmt.Errorf("no base repository present")
		}
	}

	_, err := factory.HttpClient()
	require.Nil(t, err)

	cmd := NewCmdConfigCompile(factory)
	return cmdtest.ExecuteCommand(cmd, cli, stdout, stderr)
}
