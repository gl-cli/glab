package artifact

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

const numTestFiles = 100

func createTestZipFile() (*os.File, error) {
	tempFile, err := os.CreateTemp("", "temp-*.zip")
	if err != nil {
		return nil, err
	}

	zipWriter := zip.NewWriter(tempFile)

	for i := 0; i < numTestFiles; i++ {
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

	err = readZip(reader, targetDir, defaultZIPReadLimit, defaultZIPFileLimit)
	require.NoError(t, err)

	files, err := listFilesInDir(targetDir)
	require.NoError(t, err)
	require.Len(t, files, numTestFiles)

	for _, file := range files {
		content, err := os.ReadFile(filepath.Join(targetDir, file))
		require.NoError(t, err)
		require.Equal(t, file, string(content))
	}
}

func TestFileLimitExceeded(t *testing.T) {
	zip, err := createTestZipFile()
	require.NoError(t, err)
	defer os.Remove(zip.Name())

	reader, err := toByteReader(zip.Name())
	require.NoError(t, err)

	err = readZip(reader, os.TempDir(), int64(defaultZIPReadLimit), 50)
	require.Error(t, err)
	require.Contains(t, err.Error(), "zip archive includes too many files")
}

func TestReadLimitExceeded(t *testing.T) {
	zip, err := createTestZipFile()
	require.NoError(t, err)
	defer os.Remove(zip.Name())

	reader, err := toByteReader(zip.Name())
	require.NoError(t, err)

	err = readZip(reader, os.TempDir(), 50, defaultZIPFileLimit)
	require.Error(t, err)
	require.Contains(t, err.Error(), "extracted zip too large")
}
