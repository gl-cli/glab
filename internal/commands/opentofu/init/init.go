package init

import (
	"context"
	"fmt"
	"io"
	"os/exec"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"

	gitlab "gitlab.com/gitlab-org/api/client-go"

	"gitlab.com/gitlab-org/cli/internal/api"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/internal/iostreams"
	"gitlab.com/gitlab-org/cli/internal/mcpannotations"
)

type options struct {
	io         *iostreams.IOStreams
	baseRepo   func() (glrepo.Interface, error)
	apiClient  func(repoHost string) (*api.Client, error)
	runCommand RunCommandFunc

	stateName string
	binary    string
	directory string
	initArgs  []string
}

type RunCommandFunc func(stdout, stderr io.Writer, binary string, args []string) error

func NewCmd(f cmdutils.Factory, runCommand RunCommandFunc) *cobra.Command {
	opts := &options{
		io:         f.IO(),
		baseRepo:   f.BaseRepo,
		apiClient:  f.ApiClient,
		runCommand: runCommand,
	}

	cmd := &cobra.Command{
		Use:   "init <state> [flags]",
		Short: `Initialize OpenTofu or Terraform.`,
		Example: heredoc.Doc(`
			# Initialize state with name production in working directory
			$ glab opentofu init production

			# Initialize state with name production in infra/ directory
			$ glab opentofu init production -d infra/

			# Initialize state with name production with Terraform
			$ glab opentofu init production -b terraform

			# Initialize state with name production with reconfiguring state
			$ glab opentofu init production -- -reconfigure
		`),
		Args: cobra.MinimumNArgs(1),
		Annotations: map[string]string{
			mcpannotations.Destructive: "true",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.complete(args)

			return opts.run(cmd.Context())
		},
	}

	fl := cmd.Flags()
	fl.StringVarP(&opts.binary, "binary", "b", "tofu", "Name or path of the OpenTofu or Terraform binary to use for the initialization.")
	fl.StringVarP(&opts.directory, "directory", "d", ".", "Directory of the OpenTofu or Terraform project to initialize.")

	return cmd
}

func (o *options) complete(args []string) {
	o.stateName = args[0]
	o.initArgs = args[1:]
}

func (o *options) run(ctx context.Context) error {
	repo, err := o.baseRepo()
	if err != nil {
		return err
	}

	apiClient, err := o.apiClient(repo.RepoHost())
	if err != nil {
		return err
	}

	client := apiClient.Lab()
	baseURL := client.BaseURL()
	stateAPIURL := baseURL.JoinPath("projects", gitlab.PathEscape(repo.FullName()), "terraform", "state", o.stateName)
	args := []string{
		fmt.Sprintf(`-chdir=%s`, o.directory),
		"init",
		fmt.Sprintf(`-backend-config=address=%s`, stateAPIURL.String()),
		fmt.Sprintf(`-backend-config=lock_address=%s`, stateAPIURL.JoinPath("lock").String()),
		fmt.Sprintf(`-backend-config=unlock_address=%s`, stateAPIURL.JoinPath("lock").String()),
		`-backend-config=lock_method=POST`,
		`-backend-config=unlock_method=DELETE`,
		`-backend-config=retry_wait_min=5`,
	}

	switch ts := apiClient.AuthSource().(type) {
	case gitlab.AccessTokenAuthSource:
		args = append(args, fmt.Sprintf(`-backend-config=headers={"Authorization" = "Bearer %s"}`, ts.Token))
	case gitlab.JobTokenAuthSource:
		args = append(args, fmt.Sprintf(`-backend-config=headers={"%s" = "%s"}`, gitlab.JobTokenHeaderName, ts.Token))
	case gitlab.OAuthTokenSource:
		ot, err := ts.TokenSource.Token()
		if err != nil {
			return fmt.Errorf("unable to retrieve access token to authenticate OpenTofu")
		}
		args = append(args, fmt.Sprintf(`-backend-config=headers={"Authorization" = "Bearer %s"}`, ot.AccessToken))
	case *gitlab.PasswordCredentialsAuthSource:
		currentUser, _, err := client.Users.CurrentUser(gitlab.WithContext(ctx))
		if err != nil {
			return fmt.Errorf("unable to retrieve current user: %w", err)
		}

		args = append(args,
			fmt.Sprintf(`-backend-config=username=%s`, currentUser.Username),
			fmt.Sprintf(`-backend-config=password=%s`, ts.Password),
		)
	default:
		return fmt.Errorf("init command does not support this authentication method: %T", ts)
	}
	args = append(args, o.initArgs...)

	return o.runCommand(o.io.StdOut, o.io.StdErr, o.binary, args)
}

func RunCommand(stdout, stderr io.Writer, binary string, args []string) error {
	cmd := exec.Command(
		binary,
		args...,
	)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}
