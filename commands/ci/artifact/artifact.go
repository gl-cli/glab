package ci

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/MakeNowJust/heredoc"
	"github.com/spf13/cobra"
	"github.com/xanzy/go-gitlab"
	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/config"
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

func sanitizeAssetName(asset string) string {
	if !strings.HasPrefix(asset, "/") {
		// Prefix the asset with "/" ensures that filepath.Clean removes all `/..`
		// See rule 4 of filepath.Clean for more information: https://pkg.go.dev/path/filepath#Clean
		asset = "/" + asset
	}
	return filepath.Clean(asset)
}

func NewCmdRun(f *cmdutils.Factory) *cobra.Command {
	jobArtifactCmd := &cobra.Command{
		Use:     "artifact <refName> <jobName> [flags]",
		Short:   `Download all artifacts from the last pipeline`,
		Aliases: []string{"push"},
		Example: heredoc.Doc(`
	glab ci artifact main build
	glab ci artifact main deploy --path="artifacts/"
	`),
		Long: ``,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, err := f.BaseRepo()
			if err != nil {
				return err
			}
			apiClient, err := f.HttpClient()
			if err != nil {
				return err
			}
			path, err := cmd.Flags().GetString("path")
			if err != nil {
				return err
			}

			artifact, err := api.DownloadArtifactJob(apiClient, repo.FullName(), args[0], &gitlab.DownloadArtifactsFileOptions{Job: &args[1]})
			if err != nil {
				return err
			}

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

			for _, v := range zipReader.File {
				sanitizedAssetName := sanitizeAssetName(v.Name)

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

					err = ensurePathIsCreated(destPath)
					if err != nil {
						return err
					}

					symlinkCheck, _ := os.Lstat(destPath)

					if symlinkCheck != nil && symlinkCheck.Mode()&os.ModeSymlink != 0 {
						return fmt.Errorf("file in artifact would overwrite a symbolic link- cannot extract")
					}

					dstFile, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, v.Mode())
					if err != nil {
						return err
					}
					if _, err := io.Copy(dstFile, srcFile); err != nil {
						return err
					}
				}
			}
			return nil
		},
	}
	jobArtifactCmd.Flags().StringP("path", "p", "./", "Path to download the artifact files")

	return jobArtifactCmd
}
