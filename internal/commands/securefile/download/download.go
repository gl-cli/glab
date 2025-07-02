package download

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"

	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
)

func NewCmdDownload(f cmdutils.Factory) *cobra.Command {
	securefileDownloadCmd := &cobra.Command{
		Use:   "download <fileID> [flags]",
		Short: `Download a secure file for a project.`,
		Example: heredoc.Doc(`
		    Download a project's secure file using the file's ID.
		    - glab securefile download 1

		    Download a project's secure file using the file's ID to a given path.
		    - glab securefile download 1 --path="securefiles/file.txt"
		`),
		Long: ``,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			apiClient, err := f.HttpClient()
			if err != nil {
				return err
			}

			repo, err := f.BaseRepo()
			if err != nil {
				return err
			}

			fileID, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("Secure file ID must be an integer: %s", args[0])
			}

			path, err := cmd.Flags().GetString("path")
			if err != nil {
				return fmt.Errorf("Unable to get path flag: %v", err)
			}

			err = saveFile(apiClient, repo, fileID, path)
			if err != nil {
				return err
			}

			fmt.Fprintln(f.IO().StdOut, "Downloaded secure file with ID", fileID)
			return nil
		},
	}
	securefileDownloadCmd.Flags().StringP("path", "p", "./downloaded.tmp", "Path to download the secure file to, including filename and extension.")
	return securefileDownloadCmd
}

func saveFile(apiClient *gitlab.Client, repo glrepo.Interface, fileID int, path string) error {
	contents, _, err := apiClient.SecureFiles.DownloadSecureFile(repo.FullName(), fileID)
	if err != nil {
		return fmt.Errorf("Error downloading secure file: %v", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("Error creating directory: %v", err)
		}
	}
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("Error creating file: %v", err)
	}

	defer func() {
		closeErr := file.Close()
		if closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	_, err = io.Copy(file, contents)
	if err != nil {
		return fmt.Errorf("Error writing to file: %v", err)
	}
	return nil
}
