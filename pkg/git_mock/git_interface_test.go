package git_mock

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestCurrentBranch(t *testing.T) {
	tests := []struct {
		name           string
		setupMock      func(*MockGitInterface)
		expectedBranch string
		expectedError  error
	}{
		{
			name: "successful branch retrieval",
			setupMock: func(m *MockGitInterface) {
				m.EXPECT().CurrentBranch().Return("main", nil)
			},
			expectedBranch: "main",
			expectedError:  nil,
		},
		{
			name: "detached head error",
			setupMock: func(m *MockGitInterface) {
				m.EXPECT().CurrentBranch().Return("", ErrNotOnAnyBranch)
			},
			expectedBranch: "",
			expectedError:  ErrNotOnAnyBranch,
		},
		{
			name: "unknown error",
			setupMock: func(m *MockGitInterface) {
				m.EXPECT().CurrentBranch().Return("", errors.New("error!"))
			},
			expectedBranch: "",
			expectedError:  errors.New("error!"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockGit := NewMockGitInterface(ctrl)

			tt.setupMock(mockGit)

			branch, err := mockGit.CurrentBranch()

			if tt.expectedError != nil {
				require.Error(t, err)
				require.Equal(t, tt.expectedError.Error(), err.Error())
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tt.expectedBranch, branch)
		})
	}
}

func TestGetDefaultBranch(t *testing.T) {
	tests := []struct {
		name           string
		remote         string
		setupMock      func(*MockGitInterface, string)
		expectedBranch string
		expectedError  error
	}{
		{
			name:   "successful default branch retrieval",
			remote: "origin",
			setupMock: func(m *MockGitInterface, remote string) {
				m.EXPECT().GetDefaultBranch(remote).Return("main", nil)
			},
			expectedBranch: "main",
			expectedError:  nil,
		},
		{
			name:   "remote show command fails",
			remote: "origin",
			setupMock: func(m *MockGitInterface, remote string) {
				m.EXPECT().GetDefaultBranch(remote).Return("", errors.New("could not find default branch"))
			},
			expectedBranch: "",
			expectedError:  errors.New("could not find default branch"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockGit := NewMockGitInterface(ctrl)

			tt.setupMock(mockGit, tt.remote)

			branch, err := mockGit.GetDefaultBranch(tt.remote)

			if tt.expectedError != nil {
				require.Error(t, err)
				require.Equal(t, tt.expectedError.Error(), err.Error())
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tt.expectedBranch, branch)
		})
	}
}

func TestRemoteBranchExists(t *testing.T) {
	tests := []struct {
		name          string
		branch        string
		setupMock     func(*MockGitInterface, string)
		expected      bool
		expectedError error
	}{
		{
			name:   "branch exists",
			branch: "feature",
			setupMock: func(m *MockGitInterface, branch string) {
				m.EXPECT().RemoteBranchExists(branch).Return(true, nil)
			},
			expected:      true,
			expectedError: nil,
		},
		{
			name:   "branch does not exist",
			branch: "BranchDoesNotExist",
			setupMock: func(m *MockGitInterface, branch string) {
				m.EXPECT().RemoteBranchExists(branch).Return(false, errors.New("could not find remote branch"))
			},
			expected:      false,
			expectedError: errors.New("could not find remote branch"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockGit := NewMockGitInterface(ctrl)

			tt.setupMock(mockGit, tt.branch)

			exists, err := mockGit.RemoteBranchExists(tt.branch)

			if tt.expectedError != nil {
				require.Error(t, err)
				require.Equal(t, tt.expectedError.Error(), err.Error())
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tt.expected, exists)
		})
	}
}

func TestDeleteLocalBranch(t *testing.T) {
	tests := []struct {
		name          string
		branch        string
		setupMock     func(*MockGitInterface, string)
		expectedError error
	}{
		{
			name:   "successful branch deletion",
			branch: "feature",
			setupMock: func(m *MockGitInterface, branch string) {
				m.EXPECT().DeleteLocalBranch(branch).Return(nil)
			},
			expectedError: nil,
		},
		{
			name:   "branch deletion fails",
			branch: "BranchDoesNotExist",
			setupMock: func(m *MockGitInterface, branch string) {
				m.EXPECT().DeleteLocalBranch(branch).Return(errors.New("could not delete local branch"))
			},
			expectedError: errors.New("could not delete local branch"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup the mock controller
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockGit := NewMockGitInterface(ctrl)

			tt.setupMock(mockGit, tt.branch)

			err := mockGit.DeleteLocalBranch(tt.branch)

			if tt.expectedError != nil {
				require.Error(t, err)
				require.Equal(t, tt.expectedError.Error(), err.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestCheckoutBranch(t *testing.T) {
	tests := []struct {
		name          string
		branch        string
		setupMock     func(*MockGitInterface, string)
		expectedError error
	}{
		{
			name:   "successful branch checkout",
			branch: "feature",
			setupMock: func(m *MockGitInterface, branch string) {
				m.EXPECT().CheckoutBranch(branch).Return(nil)
			},
			expectedError: nil,
		},
		{
			name:   "branch checkout fails",
			branch: "BranchDoesNotExist",
			setupMock: func(m *MockGitInterface, branch string) {
				m.EXPECT().CheckoutBranch(branch).Return(errors.New("could not checkout branch"))
			},
			expectedError: errors.New("could not checkout branch"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockGit := NewMockGitInterface(ctrl)

			tt.setupMock(mockGit, tt.branch)

			err := mockGit.CheckoutBranch(tt.branch)

			if tt.expectedError != nil {
				require.Error(t, err)
				require.Equal(t, tt.expectedError.Error(), err.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestNewStandardGitRunner(t *testing.T) {
	tests := []struct {
		name          string
		gitBinary     string
		expectedValue string
	}{
		{
			name:          "with provided binary path",
			gitBinary:     "/usr/local/bin/git",
			expectedValue: "/usr/local/bin/git",
		},
		{
			name:          "with empty binary path",
			gitBinary:     "",
			expectedValue: "git",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := NewStandardGitRunner(tt.gitBinary)
			require.Equal(t, tt.expectedValue, runner.gitBinary)
		})
	}
}
