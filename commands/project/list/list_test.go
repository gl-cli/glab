package list

import (
	"net/http"
	"testing"

	"gitlab.com/gitlab-org/cli/commands/cmdtest"

	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/cli/pkg/httpmock"
	"gitlab.com/gitlab-org/cli/test"
)

func runCommand(rt http.RoundTripper, isTTY bool, args string) (*test.CmdOut, error) {
	ios, _, stdout, stderr := cmdtest.InitIOStreams(isTTY, "")

	factory := cmdtest.InitFactory(ios, rt)

	_, _ = factory.HttpClient()

	cmd := NewCmdList(factory)

	return cmdtest.ExecuteCommand(cmd, args, stdout, stderr)
}

func TestProjectList(t *testing.T) {
	type httpMock struct {
		method string
		path   string
		status int
		body   string
	}

	tests := []struct {
		name        string
		httpMocks   []httpMock
		args        string
		expectedOut string
	}{
		{
			name: "when no projects are found shows an empty list",
			httpMocks: []httpMock{
				{
					http.MethodGet,
					"/api/v4/projects?order_by=last_activity_at&owned=true&page=1&per_page=30",
					http.StatusOK,
					"[]",
				},
			},
			args:        "",
			expectedOut: "Showing 0 of 0 projects (Page 0 of 0)\n\n\n",
		},
		{
			name: "when projects are found shows list",
			httpMocks: []httpMock{
				{
					http.MethodGet,
					"/api/v4/projects?order_by=last_activity_at&owned=true&page=1&per_page=30",
					http.StatusOK,
					`[{
							"id": 123,
 							"description": "This is a test project",
 							"path_with_namespace": "gitlab-org/incubation-engineering/service-desk/meta"
					}]`,
				},
			},
			args:        "",
			expectedOut: "Showing 1 of 0 projects (Page 0 of 0)\n\ngitlab-org/incubation-engineering/service-desk/meta\t\tThis is a test project\n\n",
		},
		{
			name: "when starred is passed as an arg filters by starred",
			httpMocks: []httpMock{
				{
					http.MethodGet,
					"/api/v4/projects?order_by=last_activity_at&owned=true&page=1&per_page=30&starred=true",
					http.StatusOK,
					`[{
							"id": 123,
 							"description": "This is a test project",
 							"path_with_namespace": "gitlab-org/incubation-engineering/service-desk/meta"
					}]`,
				},
			},
			args:        "--starred",
			expectedOut: "Showing 1 of 0 projects (Page 0 of 0)\n\ngitlab-org/incubation-engineering/service-desk/meta\t\tThis is a test project\n\n",
		},
		{
			name: "when member is passed as an arg filters by member",
			httpMocks: []httpMock{
				{
					http.MethodGet,
					"/api/v4/projects?membership=true&order_by=last_activity_at&owned=true&page=1&per_page=30",
					http.StatusOK,
					`[{
							"id": 123,
 							"description": "This is a test project",
 							"path_with_namespace": "gitlab-org/incubation-engineering/service-desk/meta"
					}]`,
				},
			},
			args:        "--member",
			expectedOut: "Showing 1 of 0 projects (Page 0 of 0)\n\ngitlab-org/incubation-engineering/service-desk/meta\t\tThis is a test project\n\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fakeHTTP := &httpmock.Mocker{
				MatchURL: httpmock.PathAndQuerystring,
			}
			defer fakeHTTP.Verify(t)

			for _, mock := range tc.httpMocks {
				fakeHTTP.RegisterResponder(mock.method, mock.path, httpmock.NewStringResponse(mock.status, mock.body))
			}

			output, err := runCommand(fakeHTTP, false, tc.args)

			if assert.NoErrorf(t, err, "error running command `project list %s`: %v", err) {
				out := output.String()

				assert.Equal(t, tc.expectedOut, out)
				assert.Empty(t, output.Stderr())
			}
		})
	}
}
