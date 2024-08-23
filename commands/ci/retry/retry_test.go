package retry

import (
	"net/http"
	"testing"

	"gitlab.com/gitlab-org/cli/commands/cmdtest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cli/pkg/httpmock"
	"gitlab.com/gitlab-org/cli/pkg/iostreams"
	"gitlab.com/gitlab-org/cli/test"
)

func runCommand(rt http.RoundTripper, args string) (*test.CmdOut, error) {
	ios, _, stdout, stderr := iostreams.Test()
	factory := cmdtest.InitFactory(ios, rt)

	_, _ = factory.HttpClient()

	cmd := NewCmdRetry(factory)

	return cmdtest.ExecuteCommand(cmd, args, stdout, stderr)
}

func TestCiRetry(t *testing.T) {
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
			name:        "when retry with job-id",
			args:        "1122",
			expectedOut: "Retried job (ID: 1123 ), status: pending , ref: branch-name , weburl:  https://gitlab.com/OWNER/REPO/-/jobs/1123 )\n",
			httpMocks: []httpMock{
				{
					http.MethodPost,
					"/api/v4/projects/OWNER/REPO/jobs/1122/retry",
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
			name:          "when retry with job-id throws error",
			args:          "1122",
			expectedError: "POST https://gitlab.com/api/v4/projects/OWNER/REPO/jobs/1122/retry: 403",
			expectedOut:   "",
			httpMocks: []httpMock{
				{
					http.MethodPost,
					"/api/v4/projects/OWNER/REPO/jobs/1122/retry",
					http.StatusForbidden,
					`{}`,
				},
			},
		},
		{
			name:        "when retry with job-name",
			args:        "lint -b main -p 123",
			expectedOut: "Retried job (ID: 1123 ), status: pending , ref: branch-name , weburl:  https://gitlab.com/OWNER/REPO/-/jobs/1123 )\n",
			httpMocks: []httpMock{
				{
					http.MethodPost,
					"/api/v4/projects/OWNER/REPO/jobs/1122/retry",
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
		},
		{
			name:           "when retry with job-name throws error",
			args:           "lint -b main -p 123",
			expectedError:  "list pipeline jobs: GET https://gitlab.com/api/v4/projects/OWNER/REPO/pipelines/123/jobs: 403",
			expectedStderr: "invalid job ID: lint\n",
			httpMocks: []httpMock{
				{
					http.MethodGet,
					"/api/v4/projects/OWNER%2FREPO/pipelines/123/jobs",
					http.StatusForbidden,
					`{}`,
				},
			},
		},
		{
			name:        "when retry with job-name and last pipeline",
			args:        "lint -b main",
			expectedOut: "Retried job (ID: 1123 ), status: pending , ref: branch-name , weburl:  https://gitlab.com/OWNER/REPO/-/jobs/1123 )\n",
			httpMocks: []httpMock{
				{
					http.MethodPost,
					"/api/v4/projects/OWNER/REPO/jobs/1122/retry",
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
						"last_pipeline": {
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

			output, err := runCommand(fakeHTTP, tc.args)

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
