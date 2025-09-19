package download

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"gitlab.com/gitlab-org/cli/internal/mcpannotations"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"

	"gitlab.com/gitlab-org/cli/internal/cmdutils"
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
		`),
		Long: ``,
		Args: cobra.ExactArgs(1),
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

			fileID, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("secure file ID must be an integer: %s", args[0])
			}

			path, err := cmd.Flags().GetString("path")
			if err != nil {
				return fmt.Errorf("unable to get path flag: %v", err)
			}

			noVerify, err := cmd.Flags().GetBool("no-verify")
			if err != nil {
				return fmt.Errorf("unable to get no-verify flag: %v", err)
			}

			forceDownload, err := cmd.Flags().GetBool("force-download")
			if err != nil {
				return fmt.Errorf("unable to get force-download flag: %v", err)
			}

			err = saveFile(client, f.IO().StdOut, repo.FullName(), fileID, path, !noVerify, forceDownload)
			if err != nil {
				return err
			}

			fmt.Fprintln(f.IO().StdOut, "Downloaded secure file with ID", fileID)

			return nil
		},
	}
	securefileDownloadCmd.Flags().StringP("path", "p", "./downloaded.tmp", "Path to download the secure file to, including filename and extension.")
	securefileDownloadCmd.Flags().Bool("no-verify", false, "Do not verify the checksum of the downloaded file(s). Warning: when enabled, this setting allows the download of files that are corrupt or tampered with.")
	securefileDownloadCmd.Flags().Bool("force-download", false, "Force download file(s) even if checksum verification fails. Warning: when enabled, this setting allows the download of files that are corrupt or tampered with.")

	securefileDownloadCmd.MarkFlagsMutuallyExclusive("no-verify", "force-download")

	return securefileDownloadCmd
}

func saveFile(apiClient *gitlab.Client, stdOut io.Writer, repoName string, fileID int, path string, verifyChecksum bool, forceDownload bool) (err error) {
	contents, _, err := apiClient.SecureFiles.DownloadSecureFile(repoName, fileID)
	if err != nil {
		return fmt.Errorf("error downloading secure file: %w", err)
	}

	directory := filepath.Dir(path)

	// Ensure directory exists
	dir := filepath.Dir(path)
	if dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("error creating directory: %v", err)
		}
	}

	// By default, os.CreateTemp creates temp files in the default tempfile location (os.TempDir), which may be located on a different file system or partition.
	// To prevent issues with cross-device renames, we need to ensure that the temporary file is created in the same directory as the downloaded file.
	tempFile, err := os.CreateTemp(directory, strconv.FormatInt(int64(fileID), 10))
	if err != nil {
		return fmt.Errorf("unable to create temporary file for downloaded secure file: %w", err)
	}

	defer func() {
		if closeErr := tempFile.Close(); closeErr != nil {
			if err == nil {
				err = fmt.Errorf("error closing temporary file: %w", closeErr)
			} else {
				fmt.Fprintf(stdOut, "error closing temporary file: %v\n", closeErr)
			}
		}
		if _, statErr := os.Stat(tempFile.Name()); statErr == nil { // Cleanup the temp file if it hasn't been renamed
			if removeErr := os.Remove(tempFile.Name()); removeErr != nil {
				if err == nil {
					err = fmt.Errorf("error removing temporary file: %w", removeErr)
				} else {
					fmt.Fprintf(stdOut, "error removing temporary file: %v\n", removeErr)
				}
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
				fmt.Fprintf(stdOut, "Checksum verification failed for %s: expected %s, got %s", file.Name, file.Checksum, checksum)
				fmt.Fprintln(stdOut, "\nForce-download selected, continuing to download file.")
			} else {
				return fmt.Errorf("checksum verification failed for %s: expected %s, got %s", file.Name, file.Checksum, checksum)
			}
		}
	} else {
		if _, err := io.Copy(tempFile, contents); err != nil {
			return fmt.Errorf("unable to write to downloaded file: %w", err)
		}
	}

	if err := os.Rename(tempFile.Name(), path); err != nil {
		return fmt.Errorf("unable to persist downloaded file contents: %w", err)
	}

	return err
}
