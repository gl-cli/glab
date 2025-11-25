//go:build !integration

package get

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gitlab "gitlab.com/gitlab-org/api/client-go"
	gitlabtesting "gitlab.com/gitlab-org/api/client-go/testing"

	"gitlab.com/gitlab-org/cli/internal/api"
	"gitlab.com/gitlab-org/cli/internal/testing/cmdtest"
)

func Test_GetSSHKey(t *testing.T) {
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
			Name:        "Get SSH key by ID",
			ExpectedMsg: []string{"ssh-ed25519 example"},
			cli:         "123",
			setupMock: func(tc *gitlabtesting.TestClient) {
				tc.MockUsers.EXPECT().GetSSHKey(int64(123)).Return(testKey, nil, nil)
			},
		},
		{
			Name:       "Get SSH key without ID",
			cli:        "",
			wantErr:    true,
			wantStderr: "the <key-id> argument is required when prompts are disabled.",
			setupMock:  func(tc *gitlabtesting.TestClient) {},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			// GIVEN
			testClient := gitlabtesting.NewTestClient(t)
			tc.setupMock(testClient)
			exec := cmdtest.SetupCmdForTest(
				t,
				NewCmdGet,
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
				assert.Contains(t, out.OutBuf.String(), msg)
			}
		})
	}
}
