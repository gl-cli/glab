package create

import (
	"fmt"
	"io"
	"os"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/pkg/iostreams"
)

type CreateOpts struct {
	FileName      string
	InputFilePath string

	IO         *iostreams.IOStreams
	HTTPClient func() (*gitlab.Client, error)
	BaseRepo   func() (glrepo.Interface, error)
	Config     func() (config.Config, error)
}

func NewCmdCreate(f cmdutils.Factory) *cobra.Command {
	opts := &CreateOpts{
		IO:     f.IO,
		Config: f.Config,
	}
	securefileCreateCmd := &cobra.Command{
		Use:   "create <fileName> <inputFilePath>",
		Short: `Create a new project secure file.`,
		Example: heredoc.Doc(`
			# Create a project secure file with the given name using the contents of the given path.
			$ glab securefile create "newfile.txt" "securefiles/localfile.txt"

			# Create a project secure file using the 'upload' alias.
			$ glab securefile upload "newfile.txt" "securefiles/localfile.txt"
		`),
		Long:    ``,
		Aliases: []string{"upload"},
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.HTTPClient = f.HttpClient
			opts.BaseRepo = f.BaseRepo
			opts.FileName = args[0]
			opts.InputFilePath = args[1]

			return createRun(opts)
		},
	}
	return securefileCreateCmd
}

func createRun(opts *CreateOpts) error {
	apiClient, err := opts.HTTPClient()
	if err != nil {
		return err
	}

	repo, err := opts.BaseRepo()
	if err != nil {
		return err
	}

	color := opts.IO.Color()
	opts.IO.Logf("%s Creating secure file %s=%s %s=%s\n",
		color.ProgressIcon(),
		color.Blue("repo"), repo.FullName(),
		color.Blue("fileName"), opts.FileName)

	reader, err := getReaderFromFilePath(opts.InputFilePath)
	if err != nil {
		return fmt.Errorf("Unable to read file at %s: %w", opts.InputFilePath, err)
	}

	err = api.CreateSecureFile(apiClient, repo.FullName(), opts.FileName, reader)
	if err != nil {
		return fmt.Errorf("Error creating secure file: %w", err)
	}

	opts.IO.Logf(color.Bold("%s Secure file %s created.\n"), color.GreenCheck(), opts.FileName)
	return nil
}

func getReaderFromFilePath(filePath string) (io.Reader, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}

	return file, nil
}
