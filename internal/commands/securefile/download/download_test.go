package download

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	gitlabtesting "gitlab.com/gitlab-org/api/client-go/testing"
	"gitlab.com/gitlab-org/cli/internal/testing/cmdtest"
	"gitlab.com/gitlab-org/cli/test"
)

const (
	fileContents         = "Hello"
	fileContentsChecksum = "185f8db32271fe25f561a6fc938b2e264306ec304eda518007d1764826381969"
	repoName             = "OWNER/REPO"
	fileName             = "localfile.txt"
)

func Test_SecurefileDownload(t *testing.T) {
	testCases := []struct {
		Name             string
		ExpectedMsg      []string
		expectedFileName string
		wantErr          bool
		cli              string
		wantStderr       string
		setupMocks       func(*gitlabtesting.TestClient)
	}{
		{
			Name:             "Download secure file to current folder with checksum verification",
			ExpectedMsg:      []string{"Downloaded secure file with ID 1"},
			expectedFileName: "downloaded.tmp",
			cli:              "1",
			setupMocks: func(testClient *gitlabtesting.TestClient) {
				testClient.MockSecureFiles.EXPECT().
					DownloadSecureFile(repoName, 1).
					Return(strings.NewReader(fileContents), nil, nil)
				testClient.MockSecureFiles.EXPECT().
					ShowSecureFileDetails(repoName, 1).
					Return(&gitlab.SecureFile{
						ID:       1,
						Name:     fileName,
						Checksum: fileContentsChecksum,
					}, nil, nil)
			},
		},
		{
			Name:             "Download secure file to custom folder",
			ExpectedMsg:      []string{"Downloaded secure file with ID 1"},
			expectedFileName: "newdir/new.txt",
			cli:              "1 --path=newdir/new.txt",
			setupMocks: func(testClient *gitlabtesting.TestClient) {
				testClient.MockSecureFiles.EXPECT().
					DownloadSecureFile(repoName, 1).
					Return(strings.NewReader(fileContents), nil, nil)
				testClient.MockSecureFiles.EXPECT().
					ShowSecureFileDetails(repoName, 1).
					Return(&gitlab.SecureFile{
						ID:       1,
						Name:     fileName,
						Checksum: fileContentsChecksum,
					}, nil, nil)
			},
		},
		{
			Name:             "Download secure file without checksum verification",
			ExpectedMsg:      []string{"Downloaded secure file with ID 1"},
			expectedFileName: "downloaded.tmp",
			cli:              "1 --no-verify",
			setupMocks: func(testClient *gitlabtesting.TestClient) {
				testClient.MockSecureFiles.EXPECT().
					DownloadSecureFile(repoName, 1).
					Return(strings.NewReader(fileContents), nil, nil)
			},
		},
		{
			Name:             "Download secure file with invalid checksum",
			ExpectedMsg:      []string{"checksum verification failed for localfile.txt: expected invalid_checksum, got 185f8db32271fe25f561a6fc938b2e264306ec304eda518007d1764826381969\n"},
			expectedFileName: "downloaded.tmp",
			cli:              "1",
			wantErr:          true,
			wantStderr:       "checksum verification failed for localfile.txt: expected invalid_checksum, got 185f8db32271fe25f561a6fc938b2e264306ec304eda518007d1764826381969",
			setupMocks: func(testClient *gitlabtesting.TestClient) {
				testClient.MockSecureFiles.EXPECT().
					DownloadSecureFile(repoName, 1).
					Return(strings.NewReader(fileContents), nil, nil)
				testClient.MockSecureFiles.EXPECT().
					ShowSecureFileDetails(repoName, 1).
					Return(&gitlab.SecureFile{
						ID:       1,
						Name:     fileName,
						Checksum: "invalid_checksum",
					}, nil, nil)
			},
		},
		{
			Name: "Force download secure file with invalid checksum",
			ExpectedMsg: []string{
				"Checksum verification failed for localfile.txt: expected invalid_checksum, got 185f8db32271fe25f561a6fc938b2e264306ec304eda518007d1764826381969\n",
				"Force-download selected, continuing to download file.\n",
				"Downloaded secure file with ID 1\n",
			},
			expectedFileName: "downloaded.tmp",
			cli:              "1 --force-download",
			setupMocks: func(testClient *gitlabtesting.TestClient) {
				testClient.MockSecureFiles.EXPECT().
					DownloadSecureFile(repoName, 1).
					Return(strings.NewReader(fileContents), nil, nil)
				testClient.MockSecureFiles.EXPECT().
					ShowSecureFileDetails(repoName, 1).
					Return(&gitlab.SecureFile{
						ID:       1,
						Name:     fileName,
						Checksum: "invalid_checksum",
					}, nil, nil)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			tempDir := t.TempDir()
			t.Chdir(tempDir)

			testClient := gitlabtesting.NewTestClient(t)
			tc.setupMocks(testClient)

			out, err := runCommand(testClient, tc.cli)
			if tc.wantErr {
				if assert.Error(t, err) {
					require.Equal(t, tc.wantStderr, err.Error())
				}

				// Ensure downloaded file and temp files are not persisted/cleaned up following an error
				files, err := os.ReadDir(filepath.Dir(tc.expectedFileName))
				require.NoError(t, err, "Failed to read directory")
				assert.Empty(t, files, "Directory should be empty after error")

				return
			}
			require.NoError(t, err)

			for _, msg := range tc.ExpectedMsg {
				require.Contains(t, out.String(), msg)
			}

			_, err = os.Stat(tc.expectedFileName)
			require.NoError(t, err)

			actualContent, err := os.ReadFile(tc.expectedFileName)
			require.NoError(t, err, "Failed to read downloaded test file")

			assert.Equal(t, "Hello", string(actualContent), "File content should match")
		})
	}
}

func TestSaveFile_InvalidDirectory(t *testing.T) {
	testClient := gitlabtesting.NewTestClient(t)

	testClient.MockSecureFiles.EXPECT().
		DownloadSecureFile("test/repo", 1).
		Return(nil, nil, nil)

	tempDir := t.TempDir()
	conflictFile := filepath.Join(tempDir, "conflict")
	err := os.WriteFile(conflictFile, []byte("conflict"), 0o755)
	require.NoError(t, err)
	outputPath := filepath.Join(conflictFile, "subdir", "file.txt")

	var stdout bytes.Buffer

	err = saveFile(testClient.Client, &stdout, "test/repo", 1, outputPath, false, false)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "error creating directory")
}

func runCommand(testClient *gitlabtesting.TestClient, cli string) (*test.CmdOut, error) {
	ios, _, stdout, stderr := cmdtest.TestIOStreams()
	factory := cmdtest.NewTestFactory(ios,
		cmdtest.WithGitLabClient(testClient.Client),
	)
	cmd := NewCmdDownload(factory)
	return cmdtest.ExecuteCommand(cmd, cli, stdout, stderr)
}
