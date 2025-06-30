package add

import (
	"errors"
	"io"
	"os"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	"gitlab.com/gitlab-org/cli/internal/api"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/internal/iostreams"
)

type options struct {
	apiClient func(repoHost string, cfg config.Config) (*api.Client, error)
	config    config.Config
	io        *iostreams.IOStreams

	title     string
	key       string
	expiresAt string
	usageType string

	keyFile string
}

func NewCmdAdd(f cmdutils.Factory) *cobra.Command {
	opts := &options{
		io:        f.IO(),
		apiClient: f.ApiClient,
		config:    f.Config(),
	}
	cmd := &cobra.Command{
		Use:   "add [key-file]",
		Short: "Add an SSH key to your GitLab account.",
		Long: heredoc.Doc(`
		Creates a new SSH key owned by the currently authenticated user.

		Requires the '--title' flag.
		`),
		Example: heredoc.Doc(`
			# Read ssh key from stdin and upload.
			$ glab ssh-key add -t "my title"

			# Read ssh key from specified key file, upload and set the ssh key type to "authentication".
			$ glab ssh-key add ~/.ssh/id_ed25519.pub -t "my title" --usage-type "auth"
		`),
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := opts.complete(args); err != nil {
				return err
			}

			return opts.run()
		},
	}

	cmd.Flags().StringVarP(&opts.title, "title", "t", "", "New SSH key's title.")
	cmd.Flags().StringVarP(&opts.usageType, "usage-type", "u", "auth_and_signing", "Usage scope for the key. Possible values: 'auth', 'signing' or 'auth_and_signing'. Default value: 'auth_and_signing'.")
	cmd.Flags().StringVarP(&opts.expiresAt, "expires-at", "e", "", "The expiration date of the SSH key. Uses ISO 8601 format: YYYY-MM-DDTHH:MM:SSZ.")

	_ = cmd.MarkFlagRequired("title")

	return cmd
}

func (o *options) complete(args []string) error {
	if len(args) == 0 {
		if o.io.IsOutputTTY() && o.io.IsInTTY {
			return &cmdutils.FlagError{Err: errors.New("missing key file")}
		}
		o.keyFile = "-"
	} else {
		o.keyFile = args[0]
	}

	return nil
}

func (o *options) run() error {
	c, err := o.apiClient("", o.config)
	if err != nil {
		return err
	}
	client := c.Lab()

	var keyFileReader io.Reader
	if o.keyFile == "-" {
		keyFileReader = o.io.In
		defer o.io.In.Close()
	} else {
		f, err := os.Open(o.keyFile)
		if err != nil {
			return err
		}
		defer f.Close()

		keyFileReader = f
	}

	keyInBytes, err := io.ReadAll(keyFileReader)
	if err != nil {
		return cmdutils.WrapError(err, "failed to read SSH key file.")
	}

	o.key = string(keyInBytes)

	err = UploadSSHKey(client, o.title, o.key, o.usageType, o.expiresAt)
	if err != nil {
		return cmdutils.WrapError(err, "failed to add new SSH public key.")
	}

	if o.io.IsOutputTTY() {
		cs := o.io.Color()
		o.io.Logf("%s New SSH public key added to your account.\n", cs.GreenCheck())
	}

	return nil
}
