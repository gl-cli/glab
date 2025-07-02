package create

import (
	"fmt"
	"io"
	"os"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/internal/iostreams"
)

type options struct {
	fileName      string
	inputFilePath string

	io         *iostreams.IOStreams
	httpClient func() (*gitlab.Client, error)
	baseRepo   func() (glrepo.Interface, error)
}

func NewCmdCreate(f cmdutils.Factory) *cobra.Command {
	opts := &options{
		io:         f.IO(),
		httpClient: f.HttpClient,
		baseRepo:   f.BaseRepo,
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
			opts.complete(args)

			return opts.run()
		},
	}
	return securefileCreateCmd
}

func (o *options) complete(args []string) {
	o.fileName = args[0]
	o.inputFilePath = args[1]
}

func (o *options) run() error {
	apiClient, err := o.httpClient()
	if err != nil {
		return err
	}

	repo, err := o.baseRepo()
	if err != nil {
		return err
	}

	color := o.io.Color()
	o.io.Logf("%s Creating secure file %s=%s %s=%s\n",
		color.ProgressIcon(),
		color.Blue("repo"), repo.FullName(),
		color.Blue("fileName"), o.fileName)

	reader, err := getReaderFromFilePath(o.inputFilePath)
	if err != nil {
		return fmt.Errorf("Unable to read file at %s: %w", o.inputFilePath, err)
	}

	_, _, err = apiClient.SecureFiles.CreateSecureFile(repo.FullName(), reader, &gitlab.CreateSecureFileOptions{Name: gitlab.Ptr(o.fileName)})
	if err != nil {
		return fmt.Errorf("Error creating secure file: %w", err)
	}

	o.io.Logf(color.Bold("%s Secure file %s created.\n"), color.GreenCheck(), o.fileName)
	return nil
}

func getReaderFromFilePath(filePath string) (io.Reader, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}

	return file, nil
}
