package add

import (
	"errors"
	"io"
	"os"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/pkg/iostreams"
)

type AddOpts struct {
	HTTPClient func() (*gitlab.Client, error)
	IO         *iostreams.IOStreams
	BaseRepo   func() (glrepo.Interface, error)

	Title     string
	Key       string
	ExpiresAt string
	CanPush   bool

	KeyFile string
}

func NewCmdAdd(f cmdutils.Factory) *cobra.Command {
	opts := &AddOpts{
		IO: f.IO(),
	}
	cmd := &cobra.Command{
		Use:   "add [key-file]",
		Short: "Add a deploy key to a GitLab project.",
		Long: heredoc.Doc(`
		Creates a new deploy key.

		Requires the '--title' flag.
		`),
		Example: heredoc.Doc(`
			# Read deploy key from stdin and upload.
			$ glab deploy-key add -t "my title"

			# Read deploy key from specified key file and upload
			$ cat ~/.ssh/id_ed25519.pub | glab deploy-key add --title='test' -

			# Read deploy key from specified key file, upload and set "can push" attribute.
			$ glab deploy-key add ~/.ssh/id_ed25519.pub -t "my title" --can-push true
		`),
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.HTTPClient = f.HttpClient
			opts.BaseRepo = f.BaseRepo

			if len(args) == 0 {
				if opts.IO.IsOutputTTY() && opts.IO.IsInTTY {
					return &cmdutils.FlagError{Err: errors.New("missing key file")}
				}
				opts.KeyFile = "-"
			} else {
				opts.KeyFile = args[0]
			}

			return addRun(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Title, "title", "t", "", "New deploy key's title.")
	cmd.Flags().BoolVarP(&opts.CanPush, "can-push", "c", false, "If true, deploy keys can be used for pushing code to the repository.")
	cmd.Flags().StringVarP(&opts.ExpiresAt, "expires-at", "e", "", "The expiration date of the deploy key, using the ISO-8601 format: YYYY-MM-DDTHH:MM:SSZ.")

	_ = cmd.MarkFlagRequired("title")

	return cmd
}

func addRun(opts *AddOpts) error {
	httpClient, err := opts.HTTPClient()
	if err != nil {
		return err
	}

	var keyFileReader io.Reader
	if opts.KeyFile == "-" {
		keyFileReader = opts.IO.In
		defer opts.IO.In.Close()
	} else {
		f, err := os.Open(opts.KeyFile)
		if err != nil {
			return err
		}
		defer f.Close()

		keyFileReader = f
	}

	keyInBytes, err := io.ReadAll(keyFileReader)
	if err != nil {
		return cmdutils.WrapError(err, "failed to read deploy key file.")
	}

	opts.Key = string(keyInBytes)

	baseRepo, err := opts.BaseRepo()
	if err != nil {
		return err
	}

	err = UploadDeployKey(httpClient, baseRepo.FullName(), opts.Title, opts.Key, opts.CanPush, opts.ExpiresAt)
	if err != nil {
		return cmdutils.WrapError(err, "failed to add new deploy key.")
	}

	if opts.IO.IsOutputTTY() {
		cs := opts.IO.Color()
		opts.IO.Logf("%s New deploy key added.\n", cs.GreenCheck())
	} else {
		opts.IO.Logf("New deploy key added.\n")
	}

	return nil
}
