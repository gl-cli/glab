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

	projectResponse := `[{
							"id": 123,
 							"description": "This is a test project",
 							"path_with_namespace": "gitlab-org/incubation-engineering/service-desk/meta"
					}]`
	groupResponse := `[{
							"id": 456,
							"description": "This is a test group",
							"path": "subgroup",
							"full_path": "/me/group/subgroup"
					}]`

	tests := []struct {
		name        string
		httpMock    []httpMock
		args        string
		expectedOut string
	}{
		{
			name: "when no projects are found shows an empty list",
			httpMock: []httpMock{{
				http.MethodGet,
				"/api/v4/projects?order_by=last_activity_at&owned=true&page=1&per_page=30",
				http.StatusOK,
				"[]",
			}},
			args:        "",
			expectedOut: "Showing 0 of 0 projects (Page 0 of 0).\n\n\n",
		},
		{
			name: "when no arguments, filters by ownership",
			httpMock: []httpMock{{
				http.MethodGet,
				"/api/v4/projects?order_by=last_activity_at&owned=true&page=1&per_page=30",
				http.StatusOK,
				projectResponse,
			}},
			args:        "",
			expectedOut: "Showing 1 of 0 projects (Page 0 of 0).\n\ngitlab-org/incubation-engineering/service-desk/meta\t\tThis is a test project\n\n",
		},
		{
			name: "when starred is passed as an arg, filters by starred",
			httpMock: []httpMock{{
				http.MethodGet,
				"/api/v4/projects?order_by=last_activity_at&page=1&per_page=30&starred=true",
				http.StatusOK,
				projectResponse,
			}},
			args:        "--starred",
			expectedOut: "Showing 1 of 0 projects (Page 0 of 0).\n\ngitlab-org/incubation-engineering/service-desk/meta\t\tThis is a test project\n\n",
		},
		{
			name: "when member is passed as an arg, filters by member",
			httpMock: []httpMock{{
				http.MethodGet,
				"/api/v4/projects?membership=true&order_by=last_activity_at&page=1&per_page=30",
				http.StatusOK,
				projectResponse,
			}},
			args:        "--member",
			expectedOut: "Showing 1 of 0 projects (Page 0 of 0).\n\ngitlab-org/incubation-engineering/service-desk/meta\t\tThis is a test project\n\n",
		},
		{
			name: "when mine is passed explicitly as an arg, filters by ownership",
			httpMock: []httpMock{{
				http.MethodGet,
				"/api/v4/projects?order_by=last_activity_at&owned=true&page=1&per_page=30",
				http.StatusOK,
				projectResponse,
			}},
			args:        "--mine",
			expectedOut: "Showing 1 of 0 projects (Page 0 of 0).\n\ngitlab-org/incubation-engineering/service-desk/meta\t\tThis is a test project\n\n",
		},
		{
			name: "when mine and starred are passed as args, filters by ownership and starred",
			httpMock: []httpMock{{
				http.MethodGet,
				"/api/v4/projects?order_by=last_activity_at&owned=true&page=1&per_page=30&starred=true",
				http.StatusOK,
				projectResponse,
			}},
			args:        "--mine --starred",
			expectedOut: "Showing 1 of 0 projects (Page 0 of 0).\n\ngitlab-org/incubation-engineering/service-desk/meta\t\tThis is a test project\n\n",
		},
		{
			name: "when starred and member are passed as args, filters by starred and membership",
			httpMock: []httpMock{{
				http.MethodGet,
				"/api/v4/projects?membership=true&order_by=last_activity_at&page=1&per_page=30&starred=true",
				http.StatusOK,
				projectResponse,
			}},
			args:        "--starred --member",
			expectedOut: "Showing 1 of 0 projects (Page 0 of 0).\n\ngitlab-org/incubation-engineering/service-desk/meta\t\tThis is a test project\n\n",
		},
		{
			name: "when mine and membership are passed as args, filters by ownership and membership",
			httpMock: []httpMock{{
				http.MethodGet,
				"/api/v4/projects?membership=true&order_by=last_activity_at&owned=true&page=1&per_page=30",
				http.StatusOK,
				projectResponse,
			}},
			args:        "--mine --member",
			expectedOut: "Showing 1 of 0 projects (Page 0 of 0).\n\ngitlab-org/incubation-engineering/service-desk/meta\t\tThis is a test project\n\n",
		},
		{
			name: "when mine, membership and starred is passed explicitly as arguments, filters by ownership, membership and starred",
			httpMock: []httpMock{{
				http.MethodGet,
				"/api/v4/projects?membership=true&order_by=last_activity_at&owned=true&page=1&per_page=30&starred=true",
				http.StatusOK,
				projectResponse,
			}},
			args:        "--mine --member --starred",
			expectedOut: "Showing 1 of 0 projects (Page 0 of 0).\n\ngitlab-org/incubation-engineering/service-desk/meta\t\tThis is a test project\n\n",
		},
		{
			name: "view all projects, no filters",
			httpMock: []httpMock{{
				http.MethodGet,
				"/api/v4/projects?order_by=last_activity_at&page=1&per_page=30",
				http.StatusOK,
				projectResponse,
			}},
			args:        "--all",
			expectedOut: "Showing 1 of 0 projects (Page 0 of 0).\n\ngitlab-org/incubation-engineering/service-desk/meta\t\tThis is a test project\n\n",
		},
		{
			name: "view all projects ordered by created_at date sorted descending",
			httpMock: []httpMock{{
				http.MethodGet,
				"/api/v4/projects?order_by=created_at&owned=true&page=1&per_page=30&sort=desc",
				http.StatusOK,
				projectResponse,
			}},
			args:        "--order created_at --sort desc",
			expectedOut: "Showing 1 of 0 projects (Page 0 of 0).\n\ngitlab-org/incubation-engineering/service-desk/meta\t\tThis is a test project\n\n",
		},
		{
			name: "view all projects in a specific group",
			httpMock: []httpMock{
				{
					http.MethodGet,
					"/api/v4/groups?search=%2Fme%2Fgroup%2Fsubgroup",
					http.StatusOK,
					groupResponse,
				},
				{
					http.MethodGet,
					"/api/v4/groups/456/projects?order_by=last_activity_at&owned=true&page=1&per_page=30",
					http.StatusOK,
					projectResponse,
				},
			},
			args:        "--group /me/group/subgroup",
			expectedOut: "Showing 1 of 0 projects (Page 0 of 0).\n\ngitlab-org/incubation-engineering/service-desk/meta\t\tThis is a test project\n\n",
		},
		{
			name: "view all projects in a specific group including subgroups",
			httpMock: []httpMock{
				{
					http.MethodGet,
					"/api/v4/groups?search=%2Fme%2Fgroup%2Fsubgroup",
					http.StatusOK,
					groupResponse,
				},
				{
					http.MethodGet,
					"/api/v4/groups/456/projects?include_subgroups=true&order_by=last_activity_at&owned=true&page=1&per_page=30",
					http.StatusOK,
					projectResponse,
				},
			},
			args:        "--group /me/group/subgroup --include-subgroups",
			expectedOut: "Showing 1 of 0 projects (Page 0 of 0).\n\ngitlab-org/incubation-engineering/service-desk/meta\t\tThis is a test project\n\n",
		},
		{
			name: "view all not archived projects in a specific group",
			httpMock: []httpMock{
				{
					http.MethodGet,
					"/api/v4/groups?search=%2Fme%2Fgroup%2Fsubgroup",
					http.StatusOK,
					groupResponse,
				},
				{
					http.MethodGet,
					"/api/v4/groups/456/projects?archived=false&order_by=last_activity_at&owned=true&page=1&per_page=30",
					http.StatusOK,
					projectResponse,
				},
			},
			args:        "--group /me/group/subgroup --archived=false",
			expectedOut: "Showing 1 of 0 projects (Page 0 of 0).\n\ngitlab-org/incubation-engineering/service-desk/meta\t\tThis is a test project\n\n",
		},
		{
			name: "view all archived projects in a specific group",
			httpMock: []httpMock{
				{
					http.MethodGet,
					"/api/v4/groups?search=%2Fme%2Fgroup%2Fsubgroup",
					http.StatusOK,
					groupResponse,
				},
				{
					http.MethodGet,
					"/api/v4/groups/456/projects?archived=true&order_by=last_activity_at&owned=true&page=1&per_page=30",
					http.StatusOK,
					projectResponse,
				},
			},
			args:        "--group /me/group/subgroup --archived=true",
			expectedOut: "Showing 1 of 0 projects (Page 0 of 0).\n\ngitlab-org/incubation-engineering/service-desk/meta\t\tThis is a test project\n\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fakeHTTP := &httpmock.Mocker{
				MatchURL: httpmock.PathAndQuerystring,
			}
			defer fakeHTTP.Verify(t)

			for _, mock := range tc.httpMock {
				fakeHTTP.RegisterResponder(mock.method, mock.path,
					httpmock.NewStringResponse(mock.status, mock.body))
			}

			output, err := runCommand(fakeHTTP, false, tc.args)

			if assert.NoErrorf(t, err, "error running command `project list %s`: %v", tc.args, err) {
				out := output.String()

				assert.Equal(t, tc.expectedOut, out)
				assert.Empty(t, output.Stderr())
			}
		})
	}
}
