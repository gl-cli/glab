package artifact

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/xanzy/go-gitlab"

	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/pkg/utils"
)

const (
	// Read limit is 4GB
	defaultZIPReadLimit int64 = 4 * 1024 * 1024 * 1024
	defaultZIPFileLimit int   = 100000
)

func ensurePathIsCreated(filename string) error {
	dir, _ := filepath.Split(filename)

	if _, err := os.Stat(filename); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0o700) // Create your file
		if err != nil {
			return fmt.Errorf("could not create new path: %v", err)
		}
	}
	return nil
}

func readZip(artifact *bytes.Reader, path string, zipReadLimit int64, zipFileLimit int) error {
	zipReader, err := zip.NewReader(artifact, artifact.Size())
	if err != nil {
		return err
	}

	if !config.CheckPathExists(path) {
		if err := os.Mkdir(path, 0o755); err != nil {
			return err
		}
	}

	if !strings.HasSuffix(path, "/") {
		path = path + "/"
	}

	var written int64 = 0
	if len(zipReader.File) > zipFileLimit {
		return fmt.Errorf("zip archive includes too many files: limit is %d files", zipFileLimit)
	}

	for _, v := range zipReader.File {
		sanitizedAssetName := utils.SanitizePathName(v.Name)

		destDir, err := filepath.Abs(path)
		if err != nil {
			return fmt.Errorf("resolving absolute download directory path: %v", err)
		}
		destPath := filepath.Join(destDir, sanitizedAssetName)
		if !strings.HasPrefix(destPath, destDir) {
			return fmt.Errorf("invalid file path name")
		}

		if v.FileInfo().IsDir() {
			if err := os.Mkdir(destPath, v.Mode()); err != nil {
				return err
			}
		} else {
			srcFile, err := zipReader.Open(v.Name)
			if err != nil {
				return err
			}
			defer srcFile.Close()

			limitedReader := io.LimitReader(srcFile, zipReadLimit)

			err = ensurePathIsCreated(destPath)
			if err != nil {
				return err
			}

			symlinkCheck, _ := os.Lstat(destPath)

			if symlinkCheck != nil && symlinkCheck.Mode()&os.ModeSymlink != 0 {
				return fmt.Errorf("can't extract. A file in the artifact would overwrite a symbolic link.")
			}

			dstFile, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, v.Mode())
			if err != nil {
				return err
			}
			var writtenPerFile int64
			if writtenPerFile, err = io.Copy(dstFile, limitedReader); err != nil {
				return err
			}

			written += writtenPerFile
			if written >= zipReadLimit {
				return fmt.Errorf("extracted zip too large: limit is %d bytes", zipReadLimit)
			}
		}
	}
	return nil
}

func DownloadArtifacts(apiClient *gitlab.Client, repo glrepo.Interface, path string, refName string, jobName string) error {
	artifact, err := api.DownloadArtifactJob(apiClient, repo.FullName(), refName, &gitlab.DownloadArtifactsFileOptions{Job: &jobName})
	if err != nil {
		return err
	}

	return readZip(artifact, path, defaultZIPReadLimit, defaultZIPFileLimit)
}
