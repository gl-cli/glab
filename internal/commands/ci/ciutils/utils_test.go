//go:build !integration

package ciutils

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/survivorbat/huhtest"
	"go.uber.org/mock/gomock"

	gitlab "gitlab.com/gitlab-org/api/client-go"
	gitlabtesting "gitlab.com/gitlab-org/api/client-go/testing"

	"gitlab.com/gitlab-org/cli/internal/glinstance"
	"gitlab.com/gitlab-org/cli/internal/testing/cmdtest"
	"gitlab.com/gitlab-org/cli/internal/testing/httpmock"
)

func TestGetJobId(t *testing.T) {
	type httpMock struct {
		method         string
		path           string
		status         int
		body           string
		responseHeader http.Header
	}

	tests := []struct {
		name          string
		jobName       string
		pipelineId    int
		httpMocks     []httpMock
		responder     *huhtest.Responder
		expectedOut   int64
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
					http.Header{},
				},
			},
		}, {
			name:        "when getJobId with name and pipelineId is requested and job is found on page 2",
			jobName:     "deploy",
			pipelineId:  123,
			expectedOut: 1144,
			httpMocks: []httpMock{
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
					http.Header{
						"X-Next-Page": []string{"2"}, // is the indicator that there is a next page. And 2 is the id of the next page
					},
				},
				{
					http.MethodGet,
					"/api/v4/projects/OWNER%2FREPO/pipelines/123/jobs?page=2&per_page=20",
					http.StatusOK,
					`[{
							"id": 1133,
							"name": "test",
							"status": "failed"
						}, {
							"id": 1144,
							"name": "deploy",
							"status": "failed"
						}]`,
					http.Header{},
				},
			},
		}, {
			name:          "when getJobId with name and pipelineId is requested and listJobs throws error",
			jobName:       "lint",
			pipelineId:    123,
			expectedError: "list pipeline jobs: GET https://gitlab.com/api/v4/projects/OWNER%2FREPO/pipelines/123/jobs: 403",
			expectedOut:   0,
			httpMocks: []httpMock{
				{
					http.MethodGet,
					"/api/v4/projects/OWNER%2FREPO/pipelines/123/jobs?page=1&per_page=20",
					http.StatusForbidden,
					`{}`,
					http.Header{},
				},
			},
		}, {
			name:       "when getJobId with name and last pipeline is requested",
			jobName:    "lint",
			pipelineId: 0,
			httpMocks: []httpMock{
				{
					http.MethodGet,
					"/api/v4/projects/OWNER%2FREPO/pipelines/latest?ref=main",
					http.StatusOK,
					`{
						"id": 123
					}`,
					http.Header{},
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
					http.Header{},
				},
			},
			expectedOut: 1122,
		}, {
			name:          "when getJobId with name and last pipeline is requested and getCommits throws error",
			jobName:       "lint",
			pipelineId:    0,
			expectedError: "get pipeline: get last pipeline: GET https://gitlab.com/api/v4/projects/OWNER%2FREPO/pipelines/latest: 403",
			expectedOut:   0,
			httpMocks: []httpMock{
				{
					http.MethodGet,
					"/api/v4/projects/OWNER%2FREPO/pipelines/latest?ref=main",
					http.StatusForbidden,
					`{}`,
					http.Header{},
				},
			},
		}, {
			name:          "when getJobId with name and last pipeline is requested and getJobs throws error",
			jobName:       "lint",
			pipelineId:    0,
			expectedError: "list pipeline jobs: GET https://gitlab.com/api/v4/projects/OWNER%2FREPO/pipelines/123/jobs: 403",
			expectedOut:   0,
			httpMocks: []httpMock{
				{
					http.MethodGet,
					"/api/v4/projects/OWNER%2FREPO/pipelines/latest?ref=main",
					http.StatusOK,
					`{
						"id": 123
					}`,
					http.Header{},
				},
				{
					http.MethodGet,
					"/api/v4/projects/OWNER%2FREPO/pipelines/123/jobs?page=1&per_page=20",
					http.StatusForbidden,
					`{}`,
					http.Header{},
				},
			},
		}, {
			name:        "when getJobId with pipelineId is requested, ask for job and answer",
			jobName:     "",
			pipelineId:  123,
			expectedOut: 1122,
			responder: huhtest.NewResponder().
				AddSelect("Select pipeline job to trace:", 0),
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
					http.Header{},
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
					httpmock.NewStringResponseWithHeader(mock.status, mock.body, mock.responseHeader),
				)
			}

			factoryOpts := []cmdtest.FactoryOption{
				cmdtest.WithGitLabClient(cmdtest.NewTestApiClient(t, &http.Client{Transport: fakeHTTP}, "", glinstance.DefaultHostname).Lab()),
				cmdtest.WithBranch("main"),
			}

			if tc.responder != nil {
				factoryOpts = append(factoryOpts, cmdtest.WithResponder(t, tc.responder))
			}

			ios, _, _, _ := cmdtest.TestIOStreams()
			f := cmdtest.NewTestFactory(ios, factoryOpts...)

			client, _ := f.GitLabClient()
			repo, _ := f.BaseRepo()

			output, err := GetJobId(t.Context(), &JobInputs{
				JobName:    tc.jobName,
				PipelineId: tc.pipelineId,
				Branch:     "main",
			}, &JobOptions{
				IO:     f.IO(),
				Repo:   repo,
				Client: client,
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
	t.Parallel()

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
			t.Parallel()

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
			expectedError: "failed to find job: GET https://gitlab.com/api/v4/projects/OWNER%2FREPO/jobs/1122: 403",
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
			name:          "when traceJob is requested and trace job throws error",
			jobName:       "1122",
			expectedError: "failed to find job: GET https://gitlab.com/api/v4/projects/OWNER%2FREPO/jobs/1122/trace: 403",
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

			ios, _, _, _ := cmdtest.TestIOStreams()
			f := cmdtest.NewTestFactory(ios,
				cmdtest.WithGitLabClient(cmdtest.NewTestApiClient(t, &http.Client{Transport: fakeHTTP}, "", glinstance.DefaultHostname).Lab()),
				cmdtest.WithBranch("main"),
			)

			client, _ := f.GitLabClient()
			repo, _ := f.BaseRepo()

			err := TraceJob(t.Context(), &JobInputs{
				JobName:    tc.jobName,
				PipelineId: tc.pipelineId,
				Branch:     "main",
			}, &JobOptions{
				IO:     f.IO(),
				Repo:   repo,
				Client: client,
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

// TestGetDefaultBranch_HappyPath tests successful scenarios for GetDefaultBranch
func TestGetDefaultBranch_HappyPath(t *testing.T) {
	tests := []struct {
		name           string
		defaultBranch  string
		expectedResult string
	}{
		{
			name:           "when API returns default branch",
			defaultBranch:  "develop",
			expectedResult: "develop",
		},
		{
			name:           "when API returns main as default branch",
			defaultBranch:  "main",
			expectedResult: "main",
		},
		{
			name:           "when API returns empty default branch",
			defaultBranch:  "",
			expectedResult: "main",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			testClient := gitlabtesting.NewTestClient(t)

			project := &gitlab.Project{
				DefaultBranch: tc.defaultBranch,
			}
			testClient.MockProjects.EXPECT().GetProject("OWNER/REPO", gomock.Any()).Return(project, nil, nil)

			ios, _, _, _ := cmdtest.TestIOStreams()
			f := cmdtest.NewTestFactory(ios,
				cmdtest.WithGitLabClient(testClient.Client),
			)

			client, _ := f.GitLabClient()
			repo, _ := f.BaseRepo()
			result := GetDefaultBranch(repo, client)
			require.Equal(t, tc.expectedResult, result)
		})
	}
}

// TestGetDefaultBranch_ErrorCases tests error scenarios for GetDefaultBranch
func TestGetDefaultBranch_ErrorCases(t *testing.T) {
	tests := []struct {
		name           string
		factoryOptions []cmdtest.FactoryOption
		expectMain     bool
	}{
		{
			name:           "when BaseRepo fails",
			factoryOptions: []cmdtest.FactoryOption{cmdtest.WithBaseRepoError(assert.AnError)},
			expectMain:     true,
		},
		{
			name:           "when GitLabClient fails",
			factoryOptions: []cmdtest.FactoryOption{cmdtest.WithGitLabClientError(assert.AnError)},
			expectMain:     true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ios, _, _, _ := cmdtest.TestIOStreams()
			factory := cmdtest.NewTestFactory(ios, tc.factoryOptions...)

			client, _ := factory.GitLabClient()
			repo, _ := factory.BaseRepo()

			// Run the test with the configured factory
			result := GetDefaultBranch(repo, client)
			require.Equal(t, "main", result)
		})
	}
}

// TestGetDefaultBranch_APIErrorCases tests API failure scenarios for GetDefaultBranch
func TestGetDefaultBranch_APIErrorCases(t *testing.T) {
	t.Run("when API call fails", func(t *testing.T) {
		testClient := gitlabtesting.NewTestClient(t)

		testClient.MockProjects.EXPECT().GetProject("OWNER/REPO", gomock.Any()).Return(nil, nil, assert.AnError)

		ios, _, _, _ := cmdtest.TestIOStreams()
		f := cmdtest.NewTestFactory(ios,
			cmdtest.WithGitLabClient(testClient.Client),
		)

		client, _ := f.GitLabClient()
		repo, _ := f.BaseRepo()
		result := GetDefaultBranch(repo, client)
		require.Equal(t, "main", result)
	})
}

// TestGetBranch_HappyPath tests successful scenarios for GetBranch
func TestGetBranch_HappyPath(t *testing.T) {
	tests := []struct {
		name             string
		specifiedBranch  string
		gitBranch        string
		apiDefaultBranch string
		expectedResult   string
	}{
		{
			name:            "when branch is specified",
			specifiedBranch: "feature-branch",
			expectedResult:  "feature-branch",
		},
		{
			name:           "when no branch specified and git works",
			gitBranch:      "current-git-branch",
			expectedResult: "current-git-branch",
		},
		{
			name:             "when no branch specified and git fails, uses API default",
			apiDefaultBranch: "develop",
			expectedResult:   "develop",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			testClient := gitlabtesting.NewTestClient(t)

			if tc.apiDefaultBranch != "" {
				project := &gitlab.Project{
					DefaultBranch: tc.apiDefaultBranch,
				}
				testClient.MockProjects.EXPECT().GetProject("OWNER/REPO", gomock.Any()).Return(project, nil, nil)
			}

			ios, _, _, _ := cmdtest.TestIOStreams()
			f := cmdtest.NewTestFactory(ios,
				cmdtest.WithGitLabClient(testClient.Client),
			)

			if tc.gitBranch != "" {
				f.BranchStub = func() (string, error) {
					return tc.gitBranch, nil
				}
			} else if tc.apiDefaultBranch != "" {
				f.BranchStub = func() (string, error) {
					return "", assert.AnError
				}
			}

			client, _ := f.GitLabClient()
			repo, _ := f.BaseRepo()
			result := GetBranch(tc.specifiedBranch, f.BranchStub, repo, client)
			require.Equal(t, tc.expectedResult, result)
		})
	}
}

// TestGetBranch_ErrorFallback tests error scenarios that fallback to main
func TestGetBranch_ErrorFallback(t *testing.T) {
	t.Run("when no branch specified and git fails, falls back to main", func(t *testing.T) {
		testClient := gitlabtesting.NewTestClient(t)

		// Mock API failure
		testClient.MockProjects.EXPECT().GetProject("OWNER/REPO", gomock.Any()).Return(nil, nil, assert.AnError)

		ios, _, _, _ := cmdtest.TestIOStreams()
		f := cmdtest.NewTestFactory(ios,
			cmdtest.WithGitLabClient(testClient.Client),
		)

		f.BranchStub = func() (string, error) {
			return "", assert.AnError
		}

		client, _ := f.GitLabClient()
		repo, _ := f.BaseRepo()
		result := GetBranch("", f.BranchStub, repo, client)
		require.Equal(t, "main", result)
	})
}
