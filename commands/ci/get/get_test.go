package status

import (
	"net/http"
	"testing"

	"gitlab.com/gitlab-org/cli/commands/cmdtest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cli/pkg/httpmock"
	"gitlab.com/gitlab-org/cli/test"
)

func runCommand(rt http.RoundTripper, isTTY bool, args string) (*test.CmdOut, error) {
	ios, _, stdout, stderr := cmdtest.InitIOStreams(isTTY, "")

	factory := cmdtest.InitFactory(ios, rt)

	_, _ = factory.HttpClient()

	cmd := NewCmdGet(factory)

	return cmdtest.ExecuteCommand(cmd, args, stdout, stderr)
}

func TestCIGet(t *testing.T) {
	type httpMock struct {
		method string
		path   string
		status int
		body   string
	}

	tests := []struct {
		name        string
		args        string
		httpMocks   []httpMock
		expectedOut string
	}{
		{
			name: "when get is called on an existing pipeline",
			args: "-p=123 -b=main",
			httpMocks: []httpMock{
				{
					http.MethodGet,
					"/api/v4/projects/OWNER%2FREPO/pipelines/123",
					http.StatusOK,
					`{
								"id": 123,
								"iid": 123,
								"status": "pending",
								"source": "push",
						        "user": {
									"username": "test"
						        },
						        "created_at": "2023-10-10T00:00:00Z",
						        "started_at": "2023-10-10T00:00:00Z",
						        "updated_at": "2023-10-10T00:00:00Z"
					}`,
				},
				{
					http.MethodGet,
					"/api/v4/projects/OWNER%2FREPO/pipelines/123/jobs?per_page=100",
					http.StatusOK,
					`[]`,
				},
			},
			expectedOut: "# Pipeline:\nid:\t123\nstatus:\tpending\nsource:\tpush\nref:\t\nsha:\t\ntag:\tfalse\nyaml Errors:\t\nuser:\ttest\ncreated:\t2023-10-10 00:00:00 +0000 UTC\nstarted:\t2023-10-10 00:00:00 +0000 UTC\nupdated:\t2023-10-10 00:00:00 +0000 UTC\n\n# Jobs:\n\n",
		},
		{
			name: "when get is called on an existing pipeline with variables",
			args: "-p=456 -b=main --with-variables",
			httpMocks: []httpMock{
				{
					http.MethodGet,
					"/api/v4/projects/OWNER%2FREPO/pipelines/456",
					http.StatusOK,
					`{
								"id": 456,
								"iid": 456,
						        "project_id": 5,
								"status": "pending",
								"source": "push",
						        "user": {
									"username": "test"
						        },
						        "created_at": "2023-10-10T00:00:00Z",
						        "started_at": "2023-10-10T00:00:00Z",
						        "updated_at": "2023-10-10T00:00:00Z"
					}`,
				},
				{
					http.MethodGet,
					"/api/v4/projects/OWNER%2FREPO/pipelines/456/jobs?per_page=100",
					http.StatusOK,
					`[]`,
				},
				{
					http.MethodGet,
					"/api/v4/projects/5/pipelines/456/variables",
					http.StatusOK,
					`[{
						"key": "RUN_NIGHTLY_BUILD",
				                	"variable_type": "env_var",
									"value": "true"
					}]`,
				},
			},
			expectedOut: "# Pipeline:\nid:\t456\nstatus:\tpending\nsource:\tpush\nref:\t\nsha:\t\ntag:\tfalse\nyaml Errors:\t\nuser:\ttest\ncreated:\t2023-10-10 00:00:00 +0000 UTC\nstarted:\t2023-10-10 00:00:00 +0000 UTC\nupdated:\t2023-10-10 00:00:00 +0000 UTC\n\n# Jobs:\n\n# Variables:\nRUN_NIGHTLY_BUILD:\ttrue\n\n",
		},
		{
			name: "when get is called on an existing pipeline with variables however no variables are found",
			args: "-p=456 -b=main --with-variables",
			httpMocks: []httpMock{
				{
					http.MethodGet,
					"/api/v4/projects/OWNER%2FREPO/pipelines/456",
					http.StatusOK,
					`{
								"id": 456,
								"iid": 456,
						        "project_id": 5,
								"status": "pending",
								"source": "push",
						        "user": {
									"username": "test"
						        },
						        "created_at": "2023-10-10T00:00:00Z",
						        "started_at": "2023-10-10T00:00:00Z",
						        "updated_at": "2023-10-10T00:00:00Z"
					}`,
				},
				{
					http.MethodGet,
					"/api/v4/projects/OWNER%2FREPO/pipelines/456/jobs?per_page=100",
					http.StatusOK,
					`[]`,
				},
				{
					http.MethodGet,
					"/api/v4/projects/5/pipelines/456/variables",
					http.StatusOK,
					"[]",
				},
			},
			expectedOut: "# Pipeline:\nid:\t456\nstatus:\tpending\nsource:\tpush\nref:\t\nsha:\t\ntag:\tfalse\nyaml Errors:\t\nuser:\ttest\ncreated:\t2023-10-10 00:00:00 +0000 UTC\nstarted:\t2023-10-10 00:00:00 +0000 UTC\nupdated:\t2023-10-10 00:00:00 +0000 UTC\n\n# Jobs:\n\n# Variables:\nNo variables found in pipeline.\n",
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
			require.Nil(t, err)

			assert.Equal(t, tc.expectedOut, output.String())
			assert.Empty(t, output.Stderr())
		})
	}
}
