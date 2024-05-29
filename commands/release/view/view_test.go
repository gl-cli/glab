package view

import (
	"net/http"
	"regexp"
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

	cmd := NewCmdView(factory)

	return cmdtest.ExecuteCommand(cmd, cli, stdout, stderr)
}

func TestReleaseView(t *testing.T) {
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
	}{
		{
			name: "view release with specific tag",
			cli:  "0.0.1",
			httpMock: httpMock{
				http.MethodGet,
				"/api/v4/projects/OWNER/REPO/releases/0.0.1",
				http.StatusOK,
				"testdata/release.json",
			},
		},
		{
			name: "view latest release",
			cli:  "",
			httpMock: httpMock{
				http.MethodGet,
				"/api/v4/projects/OWNER/REPO/releases",
				http.StatusOK,
				"testdata/releases.json",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fakeHTTP := httpmock.New()
			defer fakeHTTP.Verify(t)

			fakeHTTP.RegisterResponder(tc.httpMock.method, tc.httpMock.path,
				httpmock.NewFileResponse(tc.httpMock.status, tc.httpMock.bodyFile))

			output, err := runCommand(fakeHTTP, false, tc.cli)

			out := output.String()
			timeRE := regexp.MustCompile(`\d+ years`)
			out = timeRE.ReplaceAllString(out, "X years")

			if assert.NoErrorf(t, err, "error running command `view %s`: %v", tc.cli, err) {
				assert.Equal(t, heredoc.Doc(`test_release
											Test User released this about X years ago
											26e80b26 - 0.0.1



											ASSETS
											test asset	https://gitlab.com/OWNER/REPO/-/releases/0.0.1/downloads/test_asset
											
											SOURCES
											https://gitlab.com/OWNER/REPO/-/archive/0.0.1/REPO-0.0.1.zip
											
											
											View this release on GitLab at https://gitlab.com/OWNER/REPO/-/releases/0.0.1
											`), out)
				assert.Empty(t, output.Stderr())
			}
		})
	}
}
