package delete

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
	ios, _, stdout, stderr := cmdtest.TestIOStreams(cmdtest.WithTestIOStreamsAsTTY(true))

	factory := cmdtest.NewTestFactory(ios,
		cmdtest.WithApiClient(cmdtest.NewTestApiClient(t, &http.Client{Transport: rt}, "", glinstance.DefaultHostname)),
	)

	cmd := NewCmdDelete(factory)

	return cmdtest.ExecuteCommand(cmd, cli, stdout, stderr)
}

func TestProjectDelete(t *testing.T) {
	type httpMock struct {
		method string
		path   string
		status int
		body   string
	}

	tests := []struct {
		name     string
		cli      string
		httpMock httpMock

		expectedOutput string
	}{
		{
			name: "delete my project",
			cli:  "--yes",
			httpMock: httpMock{
				http.MethodDelete,
				"/api/v4/projects/OWNER/REPO",
				http.StatusAccepted,
				`{"message":"202 Accepted"}`,
			},
			expectedOutput: "- Deleting project OWNER/REPO\n",
		},
		{
			name: "delete project",
			cli:  "foo/bar --yes",
			httpMock: httpMock{
				http.MethodDelete,
				"/api/v4/projects/foo/bar",
				http.StatusAccepted,
				`{"message":"202 Accepted"}`,
			},
			expectedOutput: "- Deleting project foo/bar\n",
		},
		{
			name: "delete group's project",
			cli:  "group/foo/bar --yes",
			httpMock: httpMock{
				http.MethodDelete,
				"/api/v4/projects/group/foo/bar",
				http.StatusAccepted,
				`{"message":"202 Accepted"}`,
			},
			expectedOutput: "- Deleting project group/foo/bar\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fakeHTTP := &httpmock.Mocker{
				MatchURL: httpmock.PathAndQuerystring,
			}
			defer fakeHTTP.Verify(t)

			fakeHTTP.RegisterResponder(tc.httpMock.method, tc.httpMock.path,
				httpmock.NewStringResponse(tc.httpMock.status, tc.httpMock.body))

			output, err := runCommand(t, fakeHTTP, tc.cli)

			if assert.NoErrorf(t, err, "error running command `project delete %s`: %v", tc.cli, err) {
				assert.Equal(t, tc.expectedOutput, output.Stderr())
				assert.Empty(t, output.String())
			}
		})
	}
}
