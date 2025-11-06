//go:build !integration

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

func createTestZipFile(t *testing.T) string {
	t.Helper()

	tempFile, err := os.CreateTemp(t.TempDir(), "temp-*.zip")
	if err != nil {
		t.Fatalf("Create temp file: %v", err)
	}

	zipWriter := zip.NewWriter(tempFile)

	for i := range numTestFiles {
		fileName := "file-" + strconv.Itoa(i) + ".txt"

		fileWriter, err := zipWriter.Create(fileName)
		if err != nil {
			t.Fatalf("Create zip: %v", err)
		}

		_, err = fileWriter.Write([]byte(fileName))
		if err != nil {
			t.Fatalf("Write zip: %v", err)
		}
	}

	if err := zipWriter.Close(); err != nil {
		t.Fatalf("Close zip: %v", err)
	}

	if err := tempFile.Close(); err != nil {
		t.Fatalf("Close temp: %v", err)
	}

	return tempFile.Name()
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
	zipName := createTestZipFile(t)

	reader, err := toByteReader(zipName)
	require.NoError(t, err)

	targetDir := t.TempDir()

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
	zipName := createTestZipFile(t)

	reader, err := toByteReader(zipName)
	require.NoError(t, err)

	err = readZip(reader, t.TempDir(), false, defaultZIPReadLimit, 50)
	require.Error(t, err)
	require.Contains(t, err.Error(), "zip archive includes too many files")
}

func TestReadLimitExceeded(t *testing.T) {
	zipName := createTestZipFile(t)

	reader, err := toByteReader(zipName)
	require.NoError(t, err)

	err = readZip(reader, t.TempDir(), false, 50, defaultZIPFileLimit)
	require.Error(t, err)
	require.Contains(t, err.Error(), "extracted zip too large")
}
