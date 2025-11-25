//go:build !integration

package status

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	gitlab "gitlab.com/gitlab-org/api/client-go"
	gitlabtesting "gitlab.com/gitlab-org/api/client-go/testing"

	"gitlab.com/gitlab-org/cli/internal/testing/cmdtest"
)

func Test_getPipelineWithFallback(t *testing.T) {
	tests := []struct {
		name           string
		setupMocks     func(*gitlabtesting.TestClient)
		branch         string
		wantPipeline   *gitlab.Pipeline
		wantErr        bool
		expectedErrMsg string
	}{
		{
			name:   "successfully gets latest pipeline",
			branch: "main",
			setupMocks: func(tc *gitlabtesting.TestClient) {
				tc.MockPipelines.EXPECT().
					GetLatestPipeline("OWNER/REPO", &gitlab.GetLatestPipelineOptions{Ref: gitlab.Ptr("main")}).
					Return(&gitlab.Pipeline{ID: 1, Status: "success"}, nil, nil)
			},
			wantPipeline: &gitlab.Pipeline{ID: 1, Status: "success"},
			wantErr:      false,
		},
		{
			name:   "falls back to MR pipeline when latest not found",
			branch: "feature",
			setupMocks: func(tc *gitlabtesting.TestClient) {
				// Latest pipeline not found
				tc.MockPipelines.EXPECT().
					GetLatestPipeline("OWNER/REPO", &gitlab.GetLatestPipelineOptions{Ref: gitlab.Ptr("feature")}).
					Return(nil, nil, errors.New("not found"))

				// Find and get MR
				tc.MockMergeRequests.EXPECT().
					ListProjectMergeRequests("OWNER/REPO", gomock.Any()).
					Return([]*gitlab.BasicMergeRequest{{IID: 1}}, nil, nil)

				tc.MockMergeRequests.EXPECT().
					GetMergeRequest("OWNER/REPO", int64(1), gomock.Any()).
					Return(&gitlab.MergeRequest{
						BasicMergeRequest: gitlab.BasicMergeRequest{IID: 1},
						HeadPipeline:      &gitlab.Pipeline{ID: 2, Status: "running"},
					}, nil, nil)

				tc.MockPipelines.EXPECT().
					GetPipeline("OWNER/REPO", int64(2), gomock.Any()).
					Return(&gitlab.Pipeline{
						ID:     2,
						Status: "running",
					}, nil, nil)
			},
			wantPipeline: &gitlab.Pipeline{ID: 2, Status: "running"},
			wantErr:      false,
		},
		{
			name:   "returns error when no pipeline found",
			branch: "feature",
			setupMocks: func(tc *gitlabtesting.TestClient) {
				// Latest pipeline not found
				tc.MockPipelines.EXPECT().
					GetLatestPipeline("OWNER/REPO", &gitlab.GetLatestPipelineOptions{Ref: gitlab.Ptr("feature")}).
					Return(nil, nil, errors.New("not found"))

				// No MRs found
				tc.MockMergeRequests.EXPECT().
					ListProjectMergeRequests("OWNER/REPO", gomock.Any()).
					Return([]*gitlab.BasicMergeRequest{}, nil, nil)
			},
			wantPipeline:   nil,
			wantErr:        true,
			expectedErrMsg: "no pipeline found for branch feature and no associated merge request found",
		},
		{
			name:   "returns error when MR has no pipeline",
			branch: "feature",
			setupMocks: func(tc *gitlabtesting.TestClient) {
				// Latest pipeline not found
				tc.MockPipelines.EXPECT().
					GetLatestPipeline("OWNER/REPO", &gitlab.GetLatestPipelineOptions{Ref: gitlab.Ptr("feature")}).
					Return(nil, nil, errors.New("not found"))

				// Find MR but no pipeline
				tc.MockMergeRequests.EXPECT().
					ListProjectMergeRequests("OWNER/REPO", gomock.Any()).
					Return([]*gitlab.BasicMergeRequest{{IID: 1}}, nil, nil)
				tc.MockMergeRequests.EXPECT().
					GetMergeRequest("OWNER/REPO", int64(1), gomock.Any()).
					Return(&gitlab.MergeRequest{
						BasicMergeRequest: gitlab.BasicMergeRequest{IID: 1},
					}, nil, nil)
			},
			wantPipeline:   nil,
			wantErr:        true,
			expectedErrMsg: "no pipeline found. It might not exist yet",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tc := gitlabtesting.NewTestClient(t)
			tt.setupMocks(tc)

			// Create a test factory with test IO streams
			ios, _, _, _ := cmdtest.TestIOStreams(cmdtest.WithTestIOStreamsAsTTY(false))
			factory := cmdtest.NewTestFactory(ios, cmdtest.WithGitLabClient(tc.Client))

			pipeline, err := getPipelineWithFallback(tc.Client, factory, "OWNER/REPO", tt.branch)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.expectedErrMsg != "" {
					assert.Contains(t, err.Error(), tt.expectedErrMsg)
				}
				assert.Nil(t, pipeline)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.wantPipeline.ID, pipeline.ID)
			assert.Equal(t, tt.wantPipeline.Status, pipeline.Status)
		})
	}
}

func TestCiStatusCommand_NoPrompt(t *testing.T) {
	// Test that the command exits cleanly when NO_PROMPT is enabled
	// and doesn't hang waiting for user input
	tc := gitlabtesting.NewTestClient(t)

	// Mock calls in expected order
	gomock.InOrder(
		// Mock a finished pipeline so the command doesn't loop
		tc.MockPipelines.EXPECT().
			GetLatestPipeline("OWNER/REPO", &gitlab.GetLatestPipelineOptions{Ref: gitlab.Ptr("main")}).
			Return(&gitlab.Pipeline{ID: 1, Status: "success"}, nil, nil),

		// Mock jobs for the pipeline - need to handle pagination
		tc.MockJobs.EXPECT().
			ListPipelineJobs("OWNER/REPO", int64(1), gomock.Any(), gomock.Any()).
			Return([]*gitlab.Job{
				{ID: 1, Name: "test", Stage: "test", Status: "success"},
			}, &gitlab.Response{NextPage: 0}, nil),
	)

	exec := cmdtest.SetupCmdForTest(t, NewCmdStatus, true,
		cmdtest.WithGitLabClient(tc.Client),
		cmdtest.WithBranch("main"),
		// Create custom option to disable prompts
		func(f *cmdtest.Factory) {
			f.IOStub.SetPrompt("true")
		},
	)

	// This should complete without hanging
	_, err := exec("")
	require.NoError(t, err)
}

func TestCiStatusCommand_WithPromptsEnabled_FinishedPipeline(t *testing.T) {
	// Test that the command shows pipeline status and exits cleanly
	// when dealing with a finished pipeline (no interactive prompts needed)
	tc := gitlabtesting.NewTestClient(t)

	// Mock calls in expected order
	gomock.InOrder(
		// Mock a finished pipeline
		tc.MockPipelines.EXPECT().
			GetLatestPipeline("OWNER/REPO", &gitlab.GetLatestPipelineOptions{Ref: gitlab.Ptr("main")}).
			Return(&gitlab.Pipeline{ID: 1, Status: "success"}, nil, nil),

		// Mock jobs for the pipeline - need to handle pagination
		tc.MockJobs.EXPECT().
			ListPipelineJobs("OWNER/REPO", int64(1), gomock.Any(), gomock.Any()).
			Return([]*gitlab.Job{
				{ID: 1, Name: "test", Stage: "test", Status: "success"},
			}, &gitlab.Response{NextPage: 0}, nil),
	)

	exec := cmdtest.SetupCmdForTest(t, NewCmdStatus, false,
		cmdtest.WithGitLabClient(tc.Client),
		cmdtest.WithBranch("main"),
	)

	// This should complete without hanging since the pipeline is finished
	_, err := exec("")
	require.NoError(t, err)
}
