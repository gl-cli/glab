//go:build !integration

package list

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	gitlabtesting "gitlab.com/gitlab-org/api/client-go/testing"
	"gitlab.com/gitlab-org/cli/internal/api"
	"gitlab.com/gitlab-org/cli/internal/testing/cmdtest"
	"go.uber.org/mock/gomock"
)

func Test_ListSSHKey(t *testing.T) {
	type testCase struct {
		Name        string
		ExpectedMsg []string
		wantErr     bool
		cli         string
		wantStderr  string
		setupMock   func(tc *gitlabtesting.TestClient)
	}

	testKey := &gitlab.SSHKey{
		ID:        123,
		Key:       "ssh-ed25519 example",
		CreatedAt: gitlab.Ptr(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
		UsageType: "auth_and_signing",
		Title:     "mysshkey",
	}

	testCases := []testCase{
		{
			Name:        "List all ssh keys",
			ExpectedMsg: []string{"Title\tKey\tUsage type\tCreated At\nmysshkey\tssh-ed25519 example\tauth_and_signing\t2025-01-01 00:00:00 +0000 UTC\n\n"},
			cli:         "",
			setupMock: func(tc *gitlabtesting.TestClient) {
				tc.MockUsers.EXPECT().ListSSHKeys(gomock.Any()).Return([]*gitlab.SSHKey{testKey}, nil, nil)
			},
		},
		{
			Name:        "When --show-id is used shows a list of keys with IDs",
			ExpectedMsg: []string{"ID\tTitle\tKey\tUsage type\tCreated At\n123\tmysshkey\tssh-ed25519 example\tauth_and_signing\t2025-01-01 00:00:00 +0000 UTC\n\n"},
			cli:         "--show-id",
			setupMock: func(tc *gitlabtesting.TestClient) {
				tc.MockUsers.EXPECT().ListSSHKeys(gomock.Any()).Return([]*gitlab.SSHKey{testKey}, nil, nil)
			},
		},
		{
			Name:        "When no keys are found returns an empty list",
			ExpectedMsg: []string{"\n"},
			cli:         "",
			setupMock: func(tc *gitlabtesting.TestClient) {
				tc.MockUsers.EXPECT().ListSSHKeys(gomock.Any()).Return([]*gitlab.SSHKey{}, nil, nil)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			// GIVEN
			testClient := gitlabtesting.NewTestClient(t)
			tc.setupMock(testClient)
			exec := cmdtest.SetupCmdForTest(
				t,
				NewCmdList,
				false,
				cmdtest.WithApiClient(cmdtest.NewTestApiClient(t, nil, "", "", api.WithGitLabClient(testClient.Client))),
			)

			// WHEN
			out, err := exec(tc.cli)

			// THEN
			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantStderr)
				return
			}
			require.NoError(t, err)
			for _, msg := range tc.ExpectedMsg {
				assert.Equal(t, out.OutBuf.String(), msg)
			}
		})
	}
}
