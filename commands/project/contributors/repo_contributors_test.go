package contributors

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

	cmd := NewCmdContributors(factory)

	return cmdtest.ExecuteCommand(cmd, cli, stdout, stderr)
}

func TestProjectContributors(t *testing.T) {
	type httpMock struct {
		method   string
		path     string
		status   int
		bodyFile string
	}

	tests := []struct {
		name     string
		cli      string
		httpMock httpMock

		expectedOutput string
	}{
		{
			name: "view project contributors",
			cli:  "",
			httpMock: httpMock{
				http.MethodGet,
				"/api/v4/projects/OWNER/REPO/repository/contributors?order_by=commits&page=1&per_page=30&sort=desc",
				http.StatusOK,
				"testdata/contributors.json",
			},
			expectedOutput: heredoc.Doc(`Showing 2 contributors on OWNER/REPO. (Page 1)

															Test User	tu@gitlab.com	41 commits
															Test User2	tu2@gitlab.com	12 commits

															`),
		},
		{
			name: "view project contributors for a different project",
			cli:  "-R foo/bar",
			httpMock: httpMock{
				http.MethodGet,
				"/api/v4/projects/foo/bar/repository/contributors?order_by=commits&page=1&per_page=30&sort=desc",
				http.StatusOK,
				"testdata/contributors.json",
			},
			expectedOutput: heredoc.Doc(`Showing 2 contributors on foo/bar. (Page 1)

															Test User	tu@gitlab.com	41 commits
															Test User2	tu2@gitlab.com	12 commits

															`),
		},
		{
			name: "view project contributors ordered by name sorted in ascending order",
			cli:  "--order name --sort asc",
			httpMock: httpMock{
				http.MethodGet,
				"/api/v4/projects/OWNER/REPO/repository/contributors?order_by=name&page=1&per_page=30&sort=asc",
				http.StatusOK,
				"testdata/contributors.json",
			},
			expectedOutput: heredoc.Doc(`Showing 2 contributors on OWNER/REPO. (Page 1)

															Test User	tu@gitlab.com	41 commits
															Test User2	tu2@gitlab.com	12 commits

															`),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fakeHTTP := &httpmock.Mocker{
				MatchURL: httpmock.PathAndQuerystring,
			}
			defer fakeHTTP.Verify(t)

			fakeHTTP.RegisterResponder(tc.httpMock.method, tc.httpMock.path,
				httpmock.NewFileResponse(tc.httpMock.status, tc.httpMock.bodyFile))

			output, err := runCommand(fakeHTTP, false, tc.cli)

			if assert.NoErrorf(t, err, "error running command `project contributors %s`: %v", tc.cli, err) {
				assert.Equal(t, tc.expectedOutput, output.String())
				assert.Empty(t, output.Stderr())
			}
		})
	}
}
