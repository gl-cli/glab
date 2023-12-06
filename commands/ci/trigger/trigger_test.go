package trigger

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

	cmd := NewCmdTrigger(factory)

	return cmdtest.ExecuteCommand(cmd, args, stdout, stderr)
}

func TestCiTrigger(t *testing.T) {
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
			name: "when trigger with job-id is created",
			args: "1122",
			httpMocks: []httpMock{
				{
					http.MethodPost,
					"/api/v4/projects/OWNER/REPO/jobs/1122/play",
					http.StatusCreated,
					`{
						"id": 1123,
						"status": "pending",
						"stage": "build",
						"name": "build-job",
						"ref": "branch-name",
						"tag": false,
						"coverage": null,
						"allow_failure": false,
						"created_at": "2022-12-01T05:13:13.703Z",
						"web_url": "https://gitlab.com/OWNER/REPO/-/jobs/1123"
					}`,
				},
			},
			expectedOut: "Triggered job (ID: 1123 ), status: pending , ref: branch-name , weburl:  https://gitlab.com/OWNER/REPO/-/jobs/1123 )\n",
		},
		{
			name: "when trigger with job-name is created",
			args: "lint -b main -p 123",
			httpMocks: []httpMock{
				{
					http.MethodPost,
					"/api/v4/projects/OWNER/REPO/jobs/1122/play",
					http.StatusCreated,
					`{
						"id": 1123,
						"status": "pending",
						"stage": "build",
						"name": "build-job",
						"ref": "branch-name",
						"tag": false,
						"coverage": null,
						"allow_failure": false,
						"created_at": "2022-12-01T05:13:13.703Z",
						"web_url": "https://gitlab.com/OWNER/REPO/-/jobs/1123"
					}`,
				},
				{
					http.MethodGet,
					"/api/v4/projects/OWNER%2FREPO/pipelines/123/jobs",
					http.StatusOK,
					`[{
							"id": 1122,
							"name": "lint",
							"status": "failed"
						}, {
							"id": 1124,
							"name": "publish",
							"status": "failed"
						}]`,
				},
			},
			expectedOut: "Triggered job (ID: 1123 ), status: pending , ref: branch-name , weburl:  https://gitlab.com/OWNER/REPO/-/jobs/1123 )\n",
		},
		{
			name: "when trigger with job-name and last pipeline is created",
			args: "lint -b main",
			httpMocks: []httpMock{
				{
					http.MethodPost,
					"/api/v4/projects/OWNER/REPO/jobs/1122/play",
					http.StatusCreated,
					`{
						"id": 1123,
						"status": "pending",
						"stage": "build",
						"name": "build-job",
						"ref": "branch-name",
						"tag": false,
						"coverage": null,
						"allow_failure": false,
						"created_at": "2022-12-01T05:13:13.703Z",
						"web_url": "https://gitlab.com/OWNER/REPO/-/jobs/1123"
					}`,
				},
				{
					http.MethodGet,
					"/api/v4/projects/OWNER%2FREPO/repository/commits/main",
					http.StatusOK,
					`{
						"last_pipeline" : {
							"id": 123
						}
					}`,
				},
				{
					http.MethodGet,
					"/api/v4/projects/OWNER%2FREPO/pipelines/123/jobs",
					http.StatusOK,
					`[{
							"id": 1122,
							"name": "lint",
							"status": "failed"
						}, {
							"id": 1124,
							"name": "publish",
							"status": "failed"
						}]`,
				},
			},
			expectedOut: "Triggered job (ID: 1123 ), status: pending , ref: branch-name , weburl:  https://gitlab.com/OWNER/REPO/-/jobs/1123 )\n",
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
