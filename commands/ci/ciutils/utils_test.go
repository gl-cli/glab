package ciutils

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/cli/commands/cmdtest"
	"gitlab.com/gitlab-org/cli/pkg/httpmock"
	"gitlab.com/gitlab-org/cli/pkg/iostreams"
	"gitlab.com/gitlab-org/cli/pkg/prompt"
)

func TestGetJobId(t *testing.T) {
	type httpMock struct {
		method string
		path   string
		status int
		body   string
	}

	tests := []struct {
		name          string
		jobName       string
		pipelineId    int
		httpMocks     []httpMock
		askOneStubs   []string
		expectedOut   int
		expectedError string
	}{
		{
			name:        "when getJobId with integer is requested",
			jobName:     "1122",
			expectedOut: 1122,
			httpMocks:   []httpMock{},
		}, {
			name:        "when getJobId with name and pipelineId is requested",
			jobName:     "lint",
			pipelineId:  123,
			expectedOut: 1122,
			httpMocks: []httpMock{
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
		}, {
			name:          "when getJobId with name and pipelineId is requested and listJobs throws error",
			jobName:       "lint",
			pipelineId:    123,
			expectedError: "list pipeline jobs: GET https://gitlab.com/api/v4/projects/OWNER/REPO/pipelines/123/jobs: 403",
			expectedOut:   0,
			httpMocks: []httpMock{
				{
					http.MethodGet,
					"/api/v4/projects/OWNER%2FREPO/pipelines/123/jobs",
					http.StatusForbidden,
					`{}`,
				},
			},
		}, {
			name:       "when getJobId with name and last pipeline is requested",
			jobName:    "lint",
			pipelineId: 0,
			httpMocks: []httpMock{
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
			expectedOut: 1122,
		}, {
			name:          "when getJobId with name and last pipeline is requested and getCommits throws error",
			jobName:       "lint",
			pipelineId:    0,
			expectedError: "get pipeline: get last pipeline: GET https://gitlab.com/api/v4/projects/OWNER/REPO/repository/commits/main: 403",
			expectedOut:   0,
			httpMocks: []httpMock{
				{
					http.MethodGet,
					"/api/v4/projects/OWNER/REPO/repository/commits/main",
					http.StatusForbidden,
					`{}`,
				},
			},
		}, {
			name:          "when getJobId with name and last pipeline is requested and getJobs throws error",
			jobName:       "lint",
			pipelineId:    0,
			expectedError: "list pipeline jobs: GET https://gitlab.com/api/v4/projects/OWNER/REPO/pipelines/123/jobs: 403",
			expectedOut:   0,
			httpMocks: []httpMock{
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
					"/api/v4/projects/OWNER%2FREPO/pipelines/123/jobs",
					http.StatusForbidden,
					`{}`,
				},
			},
		}, {
			name:        "when getJobId with pipelineId is requested, ask for job and answer",
			jobName:     "",
			pipelineId:  123,
			expectedOut: 1122,
			askOneStubs: []string{"lint (1122) - failed"},
			httpMocks: []httpMock{
				{
					http.MethodGet,
					"/api/v4/projects/OWNER%2FREPO/pipelines/123/jobs?per_page=100",
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
		}, {
			name:        "when getJobId with pipelineId is requested, ask for job and give no answer",
			jobName:     "",
			pipelineId:  123,
			expectedOut: 0,
			askOneStubs: []string{""},
			httpMocks: []httpMock{
				{
					http.MethodGet,
					"/api/v4/projects/OWNER%2FREPO/pipelines/123/jobs?per_page=100",
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

			if tc.askOneStubs != nil {
				as, restoreAsk := prompt.InitAskStubber()
				defer restoreAsk()
				for _, value := range tc.askOneStubs {
					as.StubOne(value)
				}
			}

			defer fakeHTTP.Verify(t)

			for _, mock := range tc.httpMocks {
				fakeHTTP.RegisterResponder(
					mock.method,
					mock.path,
					httpmock.NewStringResponse(mock.status, mock.body),
				)
			}

			ios, _, _, _ := iostreams.Test()
			f := cmdtest.InitFactory(ios, fakeHTTP)

			_, _ = f.HttpClient()

			apiClient, _ := f.HttpClient()
			repo, _ := f.BaseRepo()

			output, err := GetJobId(&JobInputs{
				JobName:    tc.jobName,
				PipelineId: tc.pipelineId,
				Branch:     "main",
			}, &JobOptions{
				IO:        f.IO,
				Repo:      repo,
				ApiClient: apiClient,
			})

			if tc.expectedError == "" {
				require.Nil(t, err)
			} else {
				require.NotNil(t, err)
				require.Equal(t, tc.expectedError, err.Error())
			}
			assert.Equal(t, tc.expectedOut, output)
		})
	}
}

func TestParseCSVToIntSlice(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectedOut []int
	}{
		{
			name:        "when input is empty",
			input:       "",
			expectedOut: nil,
		},
		{
			name:        "when input is a comma-separated string",
			input:       "111,222,333",
			expectedOut: []int{111, 222, 333},
		},
		{
			name:        "when input is a space-separated string",
			input:       "111 222 333 4444",
			expectedOut: []int{111, 222, 333, 4444},
		},
		{
			name:        "when input is a space-separated and comma-separated string",
			input:       "111, 222, 333, 4444",
			expectedOut: []int{111, 222, 333, 4444},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Simulate splitting raw input
			args := strings.Fields(tc.input)

			output, err := IDsFromArgs(args)
			if err != nil {
				require.Nil(t, err)
			}

			assert.Equal(t, tc.expectedOut, output)
		})
	}
}

func TestTraceJob(t *testing.T) {
	type httpMock struct {
		method string
		path   string
		status int
		body   string
	}

	tests := []struct {
		name          string
		jobName       string
		pipelineId    int
		httpMocks     []httpMock
		expectedError string
	}{
		{
			name:    "when traceJob is requested",
			jobName: "1122",
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
			name:          "when traceJob is requested and getJob throws error",
			jobName:       "1122",
			expectedError: "failed to find job: GET https://gitlab.com/api/v4/projects/OWNER/REPO/jobs/1122: 403",
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
			name:          "when traceJob is requested and getJob throws error",
			jobName:       "1122",
			expectedError: "failed to find job: GET https://gitlab.com/api/v4/projects/OWNER/REPO/jobs/1122/trace: 403",
			httpMocks: []httpMock{
				{
					http.MethodGet,
					"/api/v4/projects/OWNER/REPO/jobs/1122/trace",
					http.StatusForbidden,
					`{}`,
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
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fakeHTTP := &httpmock.Mocker{
				MatchURL: httpmock.PathAndQuerystring,
			}
			defer fakeHTTP.Verify(t)

			for _, mock := range tc.httpMocks {
				fakeHTTP.RegisterResponder(
					mock.method,
					mock.path,
					httpmock.NewStringResponse(mock.status, mock.body),
				)
			}

			ios, _, _, _ := iostreams.Test()
			f := cmdtest.InitFactory(ios, fakeHTTP)

			_, _ = f.HttpClient()

			apiClient, _ := f.HttpClient()
			repo, _ := f.BaseRepo()

			err := TraceJob(&JobInputs{
				JobName:    tc.jobName,
				PipelineId: tc.pipelineId,
				Branch:     "main",
			}, &JobOptions{
				IO:        f.IO,
				Repo:      repo,
				ApiClient: apiClient,
			})

			if tc.expectedError == "" {
				require.Nil(t, err)
			} else {
				require.NotNil(t, err)
				require.Equal(t, tc.expectedError, err.Error())
			}
		})
	}
}
