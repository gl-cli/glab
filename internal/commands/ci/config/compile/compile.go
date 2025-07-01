package compile

import (
	"fmt"
	"os"
	"strings"

	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
)

func NewCmdConfigCompile(f cmdutils.Factory) *cobra.Command {
	configCompileCmd := &cobra.Command{
		Use:   "compile",
		Short: "View the fully expanded CI/CD configuration.",
		Args:  cobra.MaximumNArgs(1),
		Example: heredoc.Doc(`
			# Uses .gitlab-ci.yml in the current directory
			$ glab ci config compile
			$ glab ci config compile .gitlab-ci.yml
			$ glab ci config compile path/to/.gitlab-ci.yml
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := ".gitlab-ci.yml"
			if len(args) == 1 {
				path = args[0]
			}
			return compileRun(f, path)
		},
	}

	configCompileCmd.SetHelpFunc(func(command *cobra.Command, strings []string) {
		// Hide "repo"-flag for this command, because it cannot be used on repositories but only on gitlab-ci files
		_ = configCompileCmd.Flags().MarkHidden("repo")

		configCompileCmd.Parent().HelpFunc()(command, strings)
	})

	return configCompileCmd
}

func compileRun(f cmdutils.Factory, path string) error {
	var err error

	apiClient, err := f.HttpClient()
	if err != nil {
		return err
	}

	repo, err := f.BaseRepo()
	if err != nil {
		return fmt.Errorf("You must be in a GitLab project repository for this action: %w", err)
	}

	project, err := repo.Project(apiClient)
	if err != nil {
		return fmt.Errorf("You must be in a GitLab project repository for this action: %w", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading CI/CD configuration at %s: %w", path, err)
	}

	compiledResult, _, err := apiClient.Validate.ProjectNamespaceLint(
		project.ID,
		&gitlab.ProjectNamespaceLintOptions{
			Content:     gitlab.Ptr(string(content)),
			DryRun:      gitlab.Ptr(false),
			Ref:         gitlab.Ptr(""),
			IncludeJobs: gitlab.Ptr(false),
		},
	)
	if err != nil {
		return err
	}

	if !compiledResult.Valid {
		errorsStr := strings.Join(compiledResult.Errors, ", ")
		return fmt.Errorf("could not compile %s: %s", path, errorsStr)
	}

	fmt.Print(compiledResult.MergedYaml)

	return nil
}
