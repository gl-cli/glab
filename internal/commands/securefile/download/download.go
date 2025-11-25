package download

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"

	gitlab "gitlab.com/gitlab-org/api/client-go"

	"gitlab.com/gitlab-org/cli/internal/api"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/mcpannotations"
)

func NewCmdDownload(f cmdutils.Factory) *cobra.Command {
	securefileDownloadCmd := &cobra.Command{
		Use:   "download <fileID> [flags]",
		Short: `Download a secure file for a project.`,
		Example: heredoc.Doc(`
		    # Download a project's secure file using the file's ID.
		    $ glab securefile download 1

		    # Download a project's secure file using the file's ID to a given path.
		    $ glab securefile download 1 --path="securefiles/file.txt"

		    # Download a project's secure file without verifying its checksum.
		    $ glab securefile download 1 --no-verify

		    # Download a project's secure file even if checksum verification fails.
		    $ glab securefile download 1 --force-download

		    # Download all (limit 100) of a project's secure files.
		    $ glab securefile download --all

		    # Download all (limit 100) of a project's secure files to a given directory.
		    $ glab securefile download --all --output-dir secure_files/
		`),
		Long: ``,
		Args: cobra.MaximumNArgs(1),
		Annotations: map[string]string{
			mcpannotations.Safe: "true",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.GitLabClient()
			if err != nil {
				return err
			}

			repo, err := f.BaseRepo()
			if err != nil {
				return err
			}

			noVerify, err := cmd.Flags().GetBool("no-verify")
			if err != nil {
				return fmt.Errorf("unable to get no-verify flag: %w", err)
			}

			forceDownload, err := cmd.Flags().GetBool("force-download")
			if err != nil {
				return fmt.Errorf("unable to get force-download flag: %w", err)
			}

			all, err := cmd.Flags().GetBool("all")
			if err != nil {
				return fmt.Errorf("unable to get all flag: %w", err)
			}

			root, err := os.OpenRoot(".")
			if err != nil {
				return fmt.Errorf("unable to open root directory: %w", err)
			}
			defer root.Close()

			if all {
				if len(args) > 0 && args[0] != "" {
					return errors.New("all flag is not compatible with arguments")
				}

				outputDir, err := cmd.Flags().GetString("output-dir")
				if err != nil {
					return fmt.Errorf("unable to get output-dir flag: %w", err)
				}

				return downloadAllSecureFiles(client, f.IO().StdOut, root, repo.FullName(), outputDir, !noVerify, forceDownload)
			} else {
				outputDirSet := cmd.Flags().Changed("output-dir")
				if outputDirSet {
					return errors.New("output-dir flag is only compatible with all flag")
				}

				if len(args) == 0 {
					return errors.New("must provide fileID argument")
				}

				fileID, err := strconv.ParseInt(args[0], 10, 64)
				if err != nil {
					return fmt.Errorf("secure file ID must be an integer: %s", args[0])
				}

				path, err := cmd.Flags().GetString("path")
				if err != nil {
					return fmt.Errorf("unable to get path flag: %w", err)
				}

				return downloadSecureFile(client, f.IO().StdOut, root, fileID, repo.FullName(), path, !noVerify, forceDownload)
			}
		},
	}
	securefileDownloadCmd.Flags().StringP("path", "p", "./downloaded.tmp", "Path to download the secure file to, including filename and extension.")
	securefileDownloadCmd.Flags().String("output-dir", ".", "Output directory for files downloaded with --all.")
	securefileDownloadCmd.Flags().Bool("no-verify", false, "Do not verify the checksum of the downloaded file(s). Warning: when enabled, this setting allows the download of files that are corrupt or tampered with.")
	securefileDownloadCmd.Flags().Bool("force-download", false, "Force download file(s) even if checksum verification fails. Warning: when enabled, this setting allows the download of files that are corrupt or tampered with.")
	securefileDownloadCmd.Flags().Bool("all", false, "Download all (limit 100) of a project's secure files. Files are downloaded with their original name and file extension.")

	securefileDownloadCmd.MarkFlagsMutuallyExclusive("no-verify", "force-download")
	securefileDownloadCmd.MarkFlagsMutuallyExclusive("path", "output-dir")
	securefileDownloadCmd.MarkFlagsMutuallyExclusive("path", "all")

	return securefileDownloadCmd
}

func downloadSecureFile(client *gitlab.Client, stdOut io.Writer, root *os.Root, fileID int64, repoName, path string, verifyChecksum, forceDownload bool) error {
	path = filepath.Clean(path)
	if err := ensureDirectoryExists(root, path); err != nil {
		return err
	}

	err := saveFile(client, stdOut, repoName, fileID, path, verifyChecksum, forceDownload)
	if err != nil {
		return err
	}

	fmt.Fprintf(stdOut, "Downloaded secure file '%s' (ID: %d)\n", filepath.Base(path), fileID)
	return nil
}

func downloadAllSecureFiles(client *gitlab.Client, stdOut io.Writer, root *os.Root, repoName, outputDir string, verifyChecksum, forceDownload bool) error {
	l := &gitlab.ListProjectSecureFilesOptions{
		ListOptions: gitlab.ListOptions{
			Page:    1,
			PerPage: api.MaxPerPage,
		},
	}

	files, _, err := client.SecureFiles.ListProjectSecureFiles(repoName, l)
	if err != nil {
		return fmt.Errorf("error fetching secure files: %w", err)
	}

	for _, file := range files {
		filePath := filepath.Join(outputDir, file.Name)

		if err := downloadSecureFile(client, stdOut, root, file.ID, repoName, filePath, verifyChecksum, forceDownload); err != nil {
			return fmt.Errorf("error downloading secure file '%s' (ID: %d): %w", file.Name, file.ID, err)
		}
	}

	return nil
}

func saveFile(apiClient *gitlab.Client, stdOut io.Writer, repoName string, fileID int64, path string, verifyChecksum, forceDownload bool) (err error) {
	contents, _, err := apiClient.SecureFiles.DownloadSecureFile(repoName, fileID)
	if err != nil {
		return fmt.Errorf("error downloading secure file: %w", err)
	}

	root, err := os.OpenRoot(".")
	if err != nil {
		return fmt.Errorf("unable to open root directory: %w", err)
	}
	defer root.Close()

	tempFile, err := createTemp(root, fileID, path)
	if err != nil {
		return fmt.Errorf("unable to create temporary file for downloaded secure file: %w", err)
	}

	defer func() {
		if closeErr := tempFile.Close(); closeErr != nil {
			closeErr = fmt.Errorf("error closing temporary file: %w", closeErr)
			err = errors.Join(err, closeErr)
		}
		if _, statErr := root.Stat(tempFile.Name()); statErr == nil { // Cleanup the temp file if it hasn't been renamed
			if removeErr := root.Remove(tempFile.Name()); removeErr != nil {
				removeErr = fmt.Errorf("error removing temporary file: %w", removeErr)
				err = errors.Join(err, removeErr)
			}
		}
	}()

	if verifyChecksum {
		file, _, err := apiClient.SecureFiles.ShowSecureFileDetails(repoName, fileID)
		if err != nil {
			return fmt.Errorf("error getting secure file details: %w", err)
		}

		hasher := sha256.New()
		teeReader := io.TeeReader(contents, hasher)

		if _, err := io.Copy(tempFile, teeReader); err != nil {
			return fmt.Errorf("unable to write to temporary file for checksum verification: %w", err)
		}

		if checksum := hex.EncodeToString((hasher.Sum(nil))); checksum != file.Checksum {
			if forceDownload {
				fmt.Fprintf(stdOut, "Checksum verification failed for %s: expected %s, got %s\n", file.Name, file.Checksum, checksum)
				fmt.Fprintln(stdOut, "Force-download selected, continuing to download file.")
			} else {
				return fmt.Errorf("checksum verification failed for %s: expected %s, got %s", file.Name, file.Checksum, checksum)
			}
		}
	} else {
		if _, err := io.Copy(tempFile, contents); err != nil {
			return fmt.Errorf("unable to write to downloaded file: %w", err)
		}
	}

	if err := root.Rename(tempFile.Name(), path); err != nil {
		return fmt.Errorf("unable to persist downloaded file contents: %w", err)
	}

	return err
}

func ensureDirectoryExists(root *os.Root, path string) error {
	dir := filepath.Dir(path)
	if dir != "." {
		if err := root.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("error creating directory: %w", err)
		}
	}

	return nil
}

// This is a modified implementation of os.CreateTemp() using root.OpenFile.
func createTemp(root *os.Root, fileID int64, path string) (*os.File, error) {
	dir := filepath.Dir(path)
	name := filepath.Join(dir, strconv.FormatInt(fileID, 10))

	// This retry logic is to handle tempfile name collisions with an existing tempfile.
	// This is probably overkill since the chances of a collision are already extremely unlikely.
	// But it is taken from the os.CreateTemp implementation, and makes a collision effectively impossible.
	try := 0
	for {
		name = name + strconv.Itoa(rand.Intn(10))
		f, err := root.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o600)
		if os.IsExist(err) {
			if try++; try < 10000 {
				continue
			}
			return nil, fmt.Errorf("failed to create tempfile after 10000 tries: %w", err)
		}
		return f, err
	}
}
