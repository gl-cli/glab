package trace

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

	cmd := NewCmdTrace(factory)

	return cmdtest.ExecuteCommand(cmd, args, stdout, stderr)
}

func TestCiTrace(t *testing.T) {
	type httpMock struct {
		method string
		path   string
		status int
		body   string
	}

	tests := []struct {
		name          string
		args          string
		httpMocks     []httpMock
		expectedOut   string
		expectedError string
	}{
		{
			name:        "when trace for job-id is requested",
			args:        "1122",
			expectedOut: "\nGetting job trace...\nShowing logs for lint job #1122.\nLorem ipsum",
			httpMocks: []httpMock{
				{
					http.MethodGet,
					"/api/v4/projects/OWNER/REPO/jobs/1122/trace",
					http.StatusOK,
					`Lorem ipsum`,
				},
				{
					http.MethodGet,
					"/api/v4/projects/OWNER/REPO/jobs/1122",
					http.StatusOK,
					`{
						"id": 1122,
						"name": "lint",
						"status": "success"
					}`,
				},
			},
		},
		{
			name:          "when trace for job-id is requested and getTrace throws error",
			args:          "1122",
			expectedError: "failed to find job: GET https://gitlab.com/api/v4/projects/OWNER/REPO/jobs/1122/trace: 403",
			expectedOut:   "\nGetting job trace...\n",
			httpMocks: []httpMock{
				{
					http.MethodGet,
					"/api/v4/projects/OWNER/REPO/jobs/1122",
					http.StatusOK,
					`{
						"id": 1122,
						"name": "lint",
						"status": "success"
					}`,
				},
				{
					http.MethodGet,
					"/api/v4/projects/OWNER/REPO/jobs/1122/trace",
					http.StatusForbidden,
					`{}`,
				},
			},
		},
		{
			name:          "when trace for job-id is requested and getJob throws error",
			args:          "1122",
			expectedError: "failed to find job: GET https://gitlab.com/api/v4/projects/OWNER/REPO/jobs/1122: 403",
			expectedOut:   "\nGetting job trace...\n",
			httpMocks: []httpMock{
				{
					http.MethodGet,
					"/api/v4/projects/OWNER/REPO/jobs/1122",
					http.StatusForbidden,
					`{}`,
				},
			},
		},
		{
			name:        "when trace for job-name is requested",
			args:        "lint -b main -p 123",
			expectedOut: "\nGetting job trace...\n",
			httpMocks: []httpMock{
				{
					http.MethodGet,
					"/api/v4/projects/OWNER/REPO/jobs/1122/trace",
					http.StatusOK,
					`Lorem ipsum`,
				},
				{
					http.MethodGet,
					"/api/v4/projects/OWNER/REPO/jobs/1122",
					http.StatusOK,
					`{
						"id": 1122,
						"name": "lint",
						"status": "success"
					}`,
				},
				{
					http.MethodGet,
					"/api/v4/projects/OWNER/REPO/pipelines/123/jobs",
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
			name:        "when trace for job-name and last pipeline is requested",
			args:        "lint -b main",
			expectedOut: "\nGetting job trace...\n",
			httpMocks: []httpMock{
				{
					http.MethodGet,
					"/api/v4/projects/OWNER/REPO/jobs/1122/trace",
					http.StatusOK,
					`Lorem ipsum`,
				},
				{
					http.MethodGet,
					"/api/v4/projects/OWNER/REPO/jobs/1122",
					http.StatusOK,
					`{
						"id": 1122,
						"name": "lint",
						"status": "success"
					}`,
				},
				{
					http.MethodGet,
					"/api/v4/projects/OWNER/REPO/repository/commits/main",
					http.StatusOK,
					`{
						"last_pipeline": {
							"id": 123
						}
					}`,
				},
				{
					http.MethodGet,
					"/api/v4/projects/OWNER/REPO/pipelines/123/jobs",
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
			assert.Empty(t, output.Stderr())
		})
	}
}
