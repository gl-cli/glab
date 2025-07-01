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

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
)

var (
	ref         string
	dryRun      bool
	includeJobs bool
)

func NewCmdLint(f cmdutils.Factory) *cobra.Command {
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
			path := ".gitlab-ci.yml"
			if len(args) == 1 {
				path = args[0]
			}
			return lintRun(f, path)
		},
	}

	pipelineCILintCmd.Flags().BoolVarP(&dryRun, "dry-run", "", false, "Run pipeline creation simulation.")
	pipelineCILintCmd.Flags().BoolVarP(&includeJobs, "include-jobs", "", false, "Response includes the list of jobs that would exist in a static check or pipeline simulation.")
	pipelineCILintCmd.Flags().StringVar(&ref, "ref", "", "When 'dry-run' is true, sets the branch or tag context for validating the CI/CD YAML configuration.")

	return pipelineCILintCmd
}

func lintRun(f cmdutils.Factory, path string) error {
	var err error
	out := f.IO().StdOut
	c := f.IO().Color()

	apiClient, err := f.HttpClient()
	if err != nil {
		return err
	}

	repo, err := f.BaseRepo()
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

	if git.IsValidURL(path) {
		resp, err := http.Get(path)
		if err != nil {
			return err
		}
		_, err = io.Copy(&stdout, resp.Body)
		if err != nil {
			return err
		}
		content = stdout.Bytes()
	} else {
		content, err = os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("%s: no such file or directory.", path)
			}
			return err
		}
	}

	fmt.Fprintln(f.IO().StdOut, "Validating...")

	lint, _, err := apiClient.Validate.ProjectNamespaceLint(
		projectID,
		&gitlab.ProjectNamespaceLintOptions{
			Content:     gitlab.Ptr(string(content)),
			DryRun:      gitlab.Ptr(dryRun),
			Ref:         gitlab.Ptr(ref),
			IncludeJobs: gitlab.Ptr(includeJobs),
		},
	)
	if err != nil {
		return err
	}

	if !lint.Valid {
		fmt.Fprintln(out, c.Red(path+" is invalid."))
		for i, err := range lint.Errors {
			i++
			fmt.Fprintln(out, i, err)
		}
		return cmdutils.SilentError
	}
	fmt.Fprintln(out, c.GreenCheck(), "CI/CD YAML is valid!")
	return nil
}
