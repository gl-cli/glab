package list

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	gitlabtesting "gitlab.com/gitlab-org/api/client-go/testing"
	"gitlab.com/gitlab-org/cli/internal/testing/cmdtest"
)

func TestGPGKeyList(t *testing.T) {
	type testCase struct {
		name        string
		cli         string
		expectedOut string
		setupMock   func(tc *gitlabtesting.TestClient)
	}

	testKey := &gitlab.GPGKey{
		ID:        1,
		Key:       "-----BEGIN PGP PUBLIC KEY BLOCK-----\n\nmQINBF...",
		CreatedAt: gitlab.Ptr(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
	}

	tests := []testCase{
		{
			name:        "when no gpg-keys are found shows an empty list",
			cli:         "",
			expectedOut: "\n",
			setupMock: func(tc *gitlabtesting.TestClient) {
				tc.MockUsers.EXPECT().ListGPGKeys().Return([]*gitlab.GPGKey{}, nil, nil)
			},
		},
		{
			name:        "when gpg-keys are found shows a list of keys",
			cli:         "",
			expectedOut: "Key\tCreated At\n-----BEGIN PGP PUBLIC KEY BLOCK-----  mQINBF...\t2025-01-01 00:00:00 +0000 UTC\n\n",
			setupMock: func(tc *gitlabtesting.TestClient) {
				tc.MockUsers.EXPECT().ListGPGKeys().Return([]*gitlab.GPGKey{testKey}, nil, nil)
			},
		},
		{
			name:        "when --show-id is used shows a list of keys with IDs",
			cli:         "--show-id",
			expectedOut: "ID\tKey\tCreated At\n1\t-----BEGIN PGP PUBLIC KEY BLOCK-----  mQINBF...\t2025-01-01 00:00:00 +0000 UTC\n\n",
			setupMock: func(tc *gitlabtesting.TestClient) {
				tc.MockUsers.EXPECT().ListGPGKeys().Return([]*gitlab.GPGKey{testKey}, nil, nil)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// GIVEN
			testClient := gitlabtesting.NewTestClient(t)
			tc.setupMock(testClient)
			exec := cmdtest.SetupCmdForTest(
				t,
				NewCmdList,
				false,
				cmdtest.WithGitLabClient(testClient.Client),
			)

			// WHEN
			out, err := exec(tc.cli)

			// THEN
			assert.NoErrorf(t, err, "error running command `gpg-key list %s`: %v", tc.cli, err)
			output := out.OutBuf.String()
			assert.Equal(t, tc.expectedOut, output)
			assert.Empty(t, out.ErrBuf.String())
		})
	}
}
