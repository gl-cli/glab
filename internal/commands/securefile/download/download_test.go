package download

import (
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
			ExpectedMsg:      []string{"Downloaded secure file 'downloaded.tmp' (ID: 1)\n"},
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
			ExpectedMsg:      []string{"Downloaded secure file 'new.txt' (ID: 1)\n"},
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
			Name:        "Download secure file to custom folder with path traversal",
			ExpectedMsg: []string{"Downloaded secure file 'new.txt' (ID: 1)\n"},
			cli:         "1 --path=../../newdir/new.txt",
			wantErr:     true,
			wantStderr:  "error creating directory: mkdirat ../../newdir: path escapes from parent",
			setupMocks:  func(testClient *gitlabtesting.TestClient) {},
		},
		{
			Name:             "Download secure file without checksum verification",
			ExpectedMsg:      []string{"Downloaded secure file 'downloaded.tmp' (ID: 1)\n"},
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
				"Downloaded secure file 'downloaded.tmp' (ID: 1)\n",
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
		{
			Name:       "Error when output-dir flag is used without all flag",
			cli:        "1 --output-dir=secure_files",
			wantErr:    true,
			wantStderr: "output-dir flag is only compatible with all flag",
			setupMocks: func(testClient *gitlabtesting.TestClient) {},
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

				if tc.expectedFileName != "" {
					// Ensure downloaded file and temp files are not persisted/cleaned up following an error
					files, err := os.ReadDir(filepath.Dir(tc.expectedFileName))
					require.NoError(t, err, "Failed to read directory")
					assert.Empty(t, files, "Directory should be empty after error")
				}

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

func Test_SecurefileDownloadAll(t *testing.T) {
	testCases := []struct {
		Name          string
		ExpectedMsg   []string
		cli           string
		wantErr       bool
		wantStderr    string
		expectedFiles []string
		setupMocks    func(*gitlabtesting.TestClient)
	}{
		{
			Name: "Download all secure files to current directory",
			ExpectedMsg: []string{
				"Downloaded secure file 'file1.txt' (ID: 1)",
				"Downloaded secure file 'file2.pdf' (ID: 2)",
			},
			cli:           "--all",
			expectedFiles: []string{"file1.txt", "file2.pdf"},
			setupMocks: func(testClient *gitlabtesting.TestClient) {
				testClient.MockSecureFiles.EXPECT().
					ListProjectSecureFiles(repoName, &gitlab.ListProjectSecureFilesOptions{
						Page:    1,
						PerPage: 100,
					}).
					Return([]*gitlab.SecureFile{
						{ID: 1, Name: "file1.txt", Checksum: fileContentsChecksum},
						{ID: 2, Name: "file2.pdf", Checksum: fileContentsChecksum},
					}, nil, nil)

				testClient.MockSecureFiles.EXPECT().
					DownloadSecureFile(repoName, 1).
					Return(strings.NewReader(fileContents), nil, nil)
				testClient.MockSecureFiles.EXPECT().
					ShowSecureFileDetails(repoName, 1).
					Return(&gitlab.SecureFile{
						ID:       1,
						Name:     "file1.txt",
						Checksum: fileContentsChecksum,
					}, nil, nil)

				testClient.MockSecureFiles.EXPECT().
					DownloadSecureFile(repoName, 2).
					Return(strings.NewReader(fileContents), nil, nil)
				testClient.MockSecureFiles.EXPECT().
					ShowSecureFileDetails(repoName, 2).
					Return(&gitlab.SecureFile{
						ID:       2,
						Name:     "file2.pdf",
						Checksum: fileContentsChecksum,
					}, nil, nil)
			},
		},
		{
			Name: "Download all secure files to custom output directory",
			ExpectedMsg: []string{
				"Downloaded secure file 'file1.txt' (ID: 1)",
				"Downloaded secure file 'file2.pdf' (ID: 2)",
			},
			cli:           "--all --output-dir=secure_files",
			expectedFiles: []string{"secure_files/file1.txt", "secure_files/file2.pdf"},
			setupMocks: func(testClient *gitlabtesting.TestClient) {
				testClient.MockSecureFiles.EXPECT().
					ListProjectSecureFiles(repoName, &gitlab.ListProjectSecureFilesOptions{
						Page:    1,
						PerPage: 100,
					}).
					Return([]*gitlab.SecureFile{
						{ID: 1, Name: "file1.txt", Checksum: fileContentsChecksum},
						{ID: 2, Name: "file2.pdf", Checksum: fileContentsChecksum},
					}, nil, nil)

				testClient.MockSecureFiles.EXPECT().
					DownloadSecureFile(repoName, 1).
					Return(strings.NewReader(fileContents), nil, nil)
				testClient.MockSecureFiles.EXPECT().
					ShowSecureFileDetails(repoName, 1).
					Return(&gitlab.SecureFile{
						ID:       1,
						Name:     "file1.txt",
						Checksum: fileContentsChecksum,
					}, nil, nil)

				testClient.MockSecureFiles.EXPECT().
					DownloadSecureFile(repoName, 2).
					Return(strings.NewReader(fileContents), nil, nil)
				testClient.MockSecureFiles.EXPECT().
					ShowSecureFileDetails(repoName, 2).
					Return(&gitlab.SecureFile{
						ID:       2,
						Name:     "file2.pdf",
						Checksum: fileContentsChecksum,
					}, nil, nil)
			},
		},
		{
			Name: "Download all secure files without checksum verification",
			ExpectedMsg: []string{
				"Downloaded secure file 'file1.txt' (ID: 1)",
			},
			cli:           "--all --no-verify",
			expectedFiles: []string{"file1.txt"},
			setupMocks: func(testClient *gitlabtesting.TestClient) {
				testClient.MockSecureFiles.EXPECT().
					ListProjectSecureFiles(repoName, &gitlab.ListProjectSecureFilesOptions{
						Page:    1,
						PerPage: 100,
					}).
					Return([]*gitlab.SecureFile{
						{ID: 1, Name: "file1.txt", Checksum: fileContentsChecksum},
					}, nil, nil)

				testClient.MockSecureFiles.EXPECT().
					DownloadSecureFile(repoName, 1).
					Return(strings.NewReader(fileContents), nil, nil)
			},
		},
		{
			Name: "Download all secure files with force download on checksum failure",
			ExpectedMsg: []string{
				"Checksum verification failed for file1.txt: expected invalid_checksum, got 185f8db32271fe25f561a6fc938b2e264306ec304eda518007d1764826381969",
				"Force-download selected, continuing to download file.",
				"Downloaded secure file 'file1.txt' (ID: 1)",
			},
			cli:           "--all --force-download",
			expectedFiles: []string{"file1.txt"},
			setupMocks: func(testClient *gitlabtesting.TestClient) {
				testClient.MockSecureFiles.EXPECT().
					ListProjectSecureFiles(repoName, &gitlab.ListProjectSecureFilesOptions{
						Page:    1,
						PerPage: 100,
					}).
					Return([]*gitlab.SecureFile{
						{ID: 1, Name: "file1.txt", Checksum: "invalid_checksum"},
					}, nil, nil)

				testClient.MockSecureFiles.EXPECT().
					DownloadSecureFile(repoName, 1).
					Return(strings.NewReader(fileContents), nil, nil)
				testClient.MockSecureFiles.EXPECT().
					ShowSecureFileDetails(repoName, 1).
					Return(&gitlab.SecureFile{
						ID:       1,
						Name:     "file1.txt",
						Checksum: "invalid_checksum",
					}, nil, nil)
			},
		},
		{
			Name:       "Error when --all flag is used with fileID argument",
			cli:        "--all 1",
			wantErr:    true,
			wantStderr: "all flag is not compatible with arguments",
			setupMocks: func(testClient *gitlabtesting.TestClient) {},
		},
		{
			Name:          "Handle empty secure files list",
			ExpectedMsg:   []string{},
			cli:           "--all",
			expectedFiles: []string{},
			setupMocks: func(testClient *gitlabtesting.TestClient) {
				testClient.MockSecureFiles.EXPECT().
					ListProjectSecureFiles(repoName, &gitlab.ListProjectSecureFilesOptions{
						Page:    1,
						PerPage: 100,
					}).
					Return([]*gitlab.SecureFile{}, nil, nil)
			},
		},
		{
			Name: "Handle secure file with path traversal",
			ExpectedMsg: []string{
				"Downloaded secure file 'passwd' (ID: 1)\n",
			},
			cli:        "--all --output-dir=../../secure_files",
			wantErr:    true,
			wantStderr: "error downloading secure file '/etc/passwd' (ID: 1): error creating directory: mkdirat ../../secure_files/etc: path escapes from parent",
			setupMocks: func(testClient *gitlabtesting.TestClient) {
				testClient.MockSecureFiles.EXPECT().
					ListProjectSecureFiles(repoName, &gitlab.ListProjectSecureFilesOptions{
						Page:    1,
						PerPage: 100,
					}).
					Return([]*gitlab.SecureFile{
						{ID: 1, Name: "/etc/passwd", Checksum: fileContentsChecksum},
					}, nil, nil)
			},
		},
		{
			Name:       "Download all secure files with checksum error",
			cli:        "--all",
			wantErr:    true,
			wantStderr: "error downloading secure file 'file2.pdf' (ID: 2): checksum verification failed for file2.pdf: expected invalid_checksum, got 185f8db32271fe25f561a6fc938b2e264306ec304eda518007d1764826381969",
			setupMocks: func(testClient *gitlabtesting.TestClient) {
				testClient.MockSecureFiles.EXPECT().
					ListProjectSecureFiles(repoName, &gitlab.ListProjectSecureFilesOptions{
						Page:    1,
						PerPage: 100,
					}).
					Return([]*gitlab.SecureFile{
						{ID: 1, Name: "file1.txt", Checksum: fileContentsChecksum},
						{ID: 2, Name: "file2.pdf", Checksum: "invalid_checksum"},
					}, nil, nil)

				testClient.MockSecureFiles.EXPECT().
					DownloadSecureFile(repoName, 1).
					Return(strings.NewReader(fileContents), nil, nil)
				testClient.MockSecureFiles.EXPECT().
					ShowSecureFileDetails(repoName, 1).
					Return(&gitlab.SecureFile{
						ID:       1,
						Name:     "file1.txt",
						Checksum: fileContentsChecksum,
					}, nil, nil)

				testClient.MockSecureFiles.EXPECT().
					DownloadSecureFile(repoName, 2).
					Return(strings.NewReader(fileContents), nil, nil)
				testClient.MockSecureFiles.EXPECT().
					ShowSecureFileDetails(repoName, 2).
					Return(&gitlab.SecureFile{
						ID:       2,
						Name:     "file2.pdf",
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
				return
			}
			require.NoError(t, err)

			for _, msg := range tc.ExpectedMsg {
				require.Contains(t, out.String(), msg)
			}

			for _, expectedFile := range tc.expectedFiles {
				_, err = os.Stat(expectedFile)
				require.NoError(t, err, "Expected file %s should exist", expectedFile)

				actualContent, err := os.ReadFile(expectedFile)
				require.NoError(t, err, "Failed to read downloaded test file %s", expectedFile)
				assert.Equal(t, "Hello", string(actualContent), "File content should match for %s", expectedFile)
			}
		})
	}
}

func TestEnsureDirectoryExists_InvalidDirectory(t *testing.T) {
	tempDir := t.TempDir()
	conflictFile := filepath.Join(tempDir, "conflict")
	err := os.WriteFile(conflictFile, []byte("conflict"), 0o755)
	require.NoError(t, err)
	outputPath := filepath.Join(conflictFile, "subdir", "file.txt")

	root, err := os.OpenRoot(".")
	require.NoError(t, err)
	defer root.Close()

	err = ensureDirectoryExists(root, outputPath)

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
