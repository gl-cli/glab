package lint

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"

	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/git"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/internal/iostreams"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
)

type options struct {
	io         *iostreams.IOStreams
	httpClient func() (*gitlab.Client, error)
	baseRepo   func() (glrepo.Interface, error)

	path        string
	ref         string
	dryRun      bool
	includeJobs bool
}

func NewCmdLint(f cmdutils.Factory) *cobra.Command {
	opts := options{
		io:         f.IO(),
		httpClient: f.HttpClient,
		baseRepo:   f.BaseRepo,
	}
	pipelineCILintCmd := &cobra.Command{
		Use:   "lint",
		Short: "Checks if your `.gitlab-ci.yml` file is valid.",
		Args:  cobra.MaximumNArgs(1),
		Example: heredoc.Doc(`
			# Uses .gitlab-ci.yml in the current directory
			$ glab ci lint
			$ glab ci lint .gitlab-ci.yml
			$ glab ci lint path/to/.gitlab-ci.yml
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.complete(args)
			return opts.run()
		},
	}

	pipelineCILintCmd.Flags().BoolVarP(&opts.dryRun, "dry-run", "", false, "Run pipeline creation simulation.")
	pipelineCILintCmd.Flags().BoolVarP(&opts.includeJobs, "include-jobs", "", false, "Response includes the list of jobs that would exist in a static check or pipeline simulation.")
	pipelineCILintCmd.Flags().StringVar(&opts.ref, "ref", "", "When 'dry-run' is true, sets the branch or tag context for validating the CI/CD YAML configuration.")

	return pipelineCILintCmd
}

func (o *options) complete(args []string) {
	if len(args) == 1 {
		o.path = args[0]
	} else {
		o.path = ".gitlab-ci.yml"
	}
}

func (o *options) run() error {
	var err error
	out := o.io.StdOut
	c := o.io.Color()

	apiClient, err := o.httpClient()
	if err != nil {
		return err
	}

	repo, err := o.baseRepo()
	if err != nil {
		return fmt.Errorf("You must be in a GitLab project repository for this action.\nError: %s", err)
	}

	project, err := repo.Project(apiClient)
	if err != nil {
		return fmt.Errorf("You must be in a GitLab project repository for this action.\nError: %s", err)
	}

	projectID := project.ID

	var content []byte
	var stdout bytes.Buffer

	if git.IsValidURL(o.path) {
		resp, err := http.Get(o.path)
		if err != nil {
			return err
		}
		_, err = io.Copy(&stdout, resp.Body)
		if err != nil {
			return err
		}
		content = stdout.Bytes()
	} else {
		content, err = os.ReadFile(o.path)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("%s: no such file or directory.", o.path)
			}
			return err
		}
	}

	fmt.Fprintln(o.io.StdOut, "Validating...")

	lint, _, err := apiClient.Validate.ProjectNamespaceLint(
		projectID,
		&gitlab.ProjectNamespaceLintOptions{
			Content:     gitlab.Ptr(string(content)),
			DryRun:      gitlab.Ptr(o.dryRun),
			Ref:         gitlab.Ptr(o.ref),
			IncludeJobs: gitlab.Ptr(o.includeJobs),
		},
	)
	if err != nil {
		return err
	}

	if !lint.Valid {
		fmt.Fprintln(out, c.Red(o.path+" is invalid."))
		for i, err := range lint.Errors {
			i++
			fmt.Fprintln(out, i, err)
		}
		return cmdutils.SilentError
	}
	fmt.Fprintln(out, c.GreenCheck(), "CI/CD YAML is valid!")
	return nil
}
