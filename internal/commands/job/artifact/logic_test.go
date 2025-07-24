package artifact

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cli/test"
)

const numTestFiles = 100

func createTestZipFile() (*os.File, error) {
	tempFile, err := os.CreateTemp("", "temp-*.zip")
	if err != nil {
		return nil, err
	}

	zipWriter := zip.NewWriter(tempFile)

	for i := range numTestFiles {
		fileName := "file-" + strconv.Itoa(i) + ".txt"

		fileWriter, err := zipWriter.Create(fileName)
		if err != nil {
			return nil, err
		}

		_, err = fileWriter.Write([]byte(fileName))
		if err != nil {
			return nil, err
		}
	}

	if err := zipWriter.Close(); err != nil {
		return nil, err
	}

	return tempFile, nil
}

func toByteReader(zipFilePath string) (*bytes.Reader, error) {
	zipBytes, err := os.ReadFile(zipFilePath)
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(zipBytes), nil
}

func listFilesInDir(dirPath string) ([]string, error) {
	var files []string
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			files = append(files, entry.Name())
		}
	}
	return files, nil
}

func TestAcceptableZipFile(t *testing.T) {
	zip, err := createTestZipFile()
	require.NoError(t, err)
	defer os.Remove(zip.Name())

	reader, err := toByteReader(zip.Name())
	require.NoError(t, err)

	targetDir, err := os.MkdirTemp("", "tempdir-*")
	require.NoError(t, err)
	defer os.RemoveAll(targetDir)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	listPaths := true
	err = readZip(reader, targetDir, listPaths, defaultZIPReadLimit, defaultZIPFileLimit)
	stdout := test.ReturnBuffer(old, r, w)
	require.NoError(t, err)

	files, err := listFilesInDir(targetDir)
	require.NoError(t, err)
	require.Len(t, files, numTestFiles)

	for _, file := range files {
		path := filepath.Join(targetDir, file)
		content, err := os.ReadFile(path)
		require.NoError(t, err)
		require.Equal(t, file, string(content))
		require.Contains(t, stdout, path)
	}
}

func TestFileLimitExceeded(t *testing.T) {
	zip, err := createTestZipFile()
	require.NoError(t, err)
	defer os.Remove(zip.Name())

	reader, err := toByteReader(zip.Name())
	require.NoError(t, err)

	err = readZip(reader, os.TempDir(), false, defaultZIPReadLimit, 50)
	require.Error(t, err)
	require.Contains(t, err.Error(), "zip archive includes too many files")
}

func TestReadLimitExceeded(t *testing.T) {
	zip, err := createTestZipFile()
	require.NoError(t, err)
	defer os.Remove(zip.Name())

	reader, err := toByteReader(zip.Name())
	require.NoError(t, err)

	err = readZip(reader, os.TempDir(), false, 50, defaultZIPFileLimit)
	require.Error(t, err)
	require.Contains(t, err.Error(), "extracted zip too large")
}
