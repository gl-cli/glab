package trigger

import (
	"net/http"
	"testing"

	"gitlab.com/gitlab-org/cli/internal/glinstance"
	"gitlab.com/gitlab-org/cli/internal/testing/cmdtest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cli/internal/testing/httpmock"
	"gitlab.com/gitlab-org/cli/test"
)

func runCommand(t *testing.T, rt http.RoundTripper, args string) (*test.CmdOut, error) {
	ios, _, stdout, stderr := cmdtest.TestIOStreams()
	factory := cmdtest.NewTestFactory(ios,
		cmdtest.WithGitLabClient(cmdtest.NewTestApiClient(t, &http.Client{Transport: rt}, "", glinstance.DefaultHostname).Lab()),
	)

	cmd := NewCmdTrigger(factory)

	return cmdtest.ExecuteCommand(cmd, args, stdout, stderr)
}

func TestCiTrigger(t *testing.T) {
	t.Parallel()

	type httpMock struct {
		method string
		path   string
		status int
		body   string
	}

	tests := []struct {
		name           string
		args           string
		httpMocks      []httpMock
		expectedError  string
		expectedStderr string
		expectedOut    string
	}{
		{
			name:        "when trigger with job-id",
			args:        "1122",
			expectedOut: "Triggered job (ID: 1123), status: pending, ref: branch-name, weburl: https://gitlab.com/OWNER/REPO/-/jobs/1123\n",
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
		},
		{
			name:          "when trigger with job-id throws error",
			args:          "1122",
			expectedError: "POST https://gitlab.com/api/v4/projects/OWNER%2FREPO/jobs/1122/play: 403",
			expectedOut:   "",
			httpMocks: []httpMock{
				{
					http.MethodPost,
					"/api/v4/projects/OWNER/REPO/jobs/1122/play",
					http.StatusForbidden,
					`{}`,
				},
			},
		},
		{
			name:        "when trigger with job-name",
			args:        "lint -b main -p 123",
			expectedOut: "Triggered job (ID: 1123), status: pending, ref: branch-name, weburl: https://gitlab.com/OWNER/REPO/-/jobs/1123\n",
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
					"/api/v4/projects/OWNER%2FREPO/pipelines/123/jobs?page=1&per_page=20",
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
		},
		{
			name:           "when trigger with job-name throws error",
			args:           "lint -b main -p 123",
			expectedError:  "list pipeline jobs: GET https://gitlab.com/api/v4/projects/OWNER%2FREPO/pipelines/123/jobs: 403",
			expectedStderr: "invalid job ID: lint\n",
			httpMocks: []httpMock{
				{
					http.MethodGet,
					"/api/v4/projects/OWNER%2FREPO/pipelines/123/jobs?page=1&per_page=20",
					http.StatusForbidden,
					`{}`,
				},
			},
		},
		{
			name:        "when trigger with job-name and last pipeline",
			args:        "lint -b main",
			expectedOut: "Triggered job (ID: 1123), status: pending, ref: branch-name, weburl: https://gitlab.com/OWNER/REPO/-/jobs/1123\n",
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
					"/api/v4/projects/OWNER%2FREPO/pipelines/latest?ref=main",
					http.StatusOK,
					`{
						"id": 123
					}`,
				},
				{
					http.MethodGet,
					"/api/v4/projects/OWNER%2FREPO/pipelines/123/jobs?page=1&per_page=20",
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

			output, err := runCommand(t, fakeHTTP, tc.args)

			if tc.expectedError == "" {
				require.Nil(t, err)
			} else {
				require.NotNil(t, err)
				require.Equal(t, tc.expectedError, err.Error())
			}

			assert.Equal(t, tc.expectedOut, output.String())
			if tc.expectedStderr != "" {
				assert.Equal(t, tc.expectedStderr, output.Stderr())
			} else {
				assert.Empty(t, output.Stderr())
			}
		})
	}
}
