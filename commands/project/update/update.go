package update

import (
	"fmt"
	"strings"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
)

const (
	defaultBranchFlag = "defaultBranch"
	descriptionFlag   = "description"
)

func NewCmdUpdate(f cmdutils.Factory) *cobra.Command {
	projectUpdateCmd := &cobra.Command{
		Use:   "update [path] [flags]",
		Short: `Update an existing GitLab project or repository.`,
		Long:  ``,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpdateProject(cmd, f, args)
		},
		Example: heredoc.Doc(`
			# Update the description for my-project.
			$ glab repo update my-project --description "This project is cool."

			# Update the default branch for my-project.
			$ glab repo update my-project --defaultBranch main
	  `),
	}

	projectUpdateCmd.Flags().String(defaultBranchFlag, "", "New default branch for the project.")
	projectUpdateCmd.Flags().StringP(descriptionFlag, "d", "", "New description for the project.")

	return projectUpdateCmd
}

func runUpdateProject(cmd *cobra.Command, f cmdutils.Factory, args []string) error {
	cfg, err := f.Config()
	if err != nil {
		return err
	}

	var projectID string
	if len(args) == 1 {
		projectID = args[0]
	}
	repo, err := getRepoFromProjectID(projectID, f)
	if err != nil {
		return err
	}

	client, err := api.NewClientWithCfg(repo.RepoHost(), cfg, false)
	if err != nil {
		return err
	}
	apiClient := client.Lab()

	opts, err := getAPIArgs(cmd.Flags())
	if err != nil {
		return err
	}

	project, err := api.UpdateProject(apiClient, repo.FullName(), opts)
	if err != nil {
		return fmt.Errorf("error updating project: %w", err)
	}

	greenCheck := f.IO().Color().Green("✓")
	fmt.Fprintf(f.IO().StdOut, "%s Updated repository %s on GitLab: %s\n", greenCheck, project.NameWithNamespace, project.WebURL)
	return nil
}

// getRepoFromProjectID wrangles the various ways of specifying a project into
// a repository object. The accepted input formats are:
//   - no project specified - assume the current dir is a Git repository and
//     look up the remote
//   - just project name - need to look up user's username and prefix it with a
//     slash
//   - user/project - can be passed verbatim to the API
//   - fully qualified GitLab URL
func getRepoFromProjectID(projectID string, f cmdutils.Factory) (glrepo.Interface, error) {
	// No project argument - use current repository
	if projectID == "" {
		// Configure client to have host of current repository
		repo, err := f.BaseRepo()
		if err != nil {
			return nil, cmdutils.WrapError(err, "`repository` is required when not running in a Git repository.")
		}
		return repo, nil
	} else {
		// If the ProjectID is a single token, use current user's namespace
		if !strings.Contains(projectID, "/") {
			apiClient, err := f.HttpClient()
			if err != nil {
				return nil, err
			}
			currentUser, err := api.CurrentUser(apiClient)
			if err != nil {
				return nil, cmdutils.WrapError(err, "Failed to retrieve your current user.")
			}

			projectID = currentUser.Username + "/" + projectID
		}

		// Get the repo full name from the ProjectID which can be a full URL or a group/repo format
		return glrepo.FromFullName(projectID)
	}
}

func getAPIArgs(flags *pflag.FlagSet) (*gitlab.EditProjectOptions, error) {
	// We need to check if no flags were defined, and if so, exit early before we
	// hit the API.
	someFlagDefined := false
	opts := &gitlab.EditProjectOptions{}

	if flags.Changed(defaultBranchFlag) {
		defaultBranch, err := flags.GetString(defaultBranchFlag)
		if err != nil {
			return nil, err
		}
		opts.DefaultBranch = &defaultBranch
		someFlagDefined = true
	}
	if flags.Changed(descriptionFlag) {
		description, err := flags.GetString(descriptionFlag)
		if err != nil {
			return nil, err
		}
		opts.Description = &description
		someFlagDefined = true
	}

	if !someFlagDefined {
		return nil, fmt.Errorf("must specify either --description or --defaultBranch")
	}
	return opts, nil
}
