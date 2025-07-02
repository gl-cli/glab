package create

import (
	"fmt"
	"net/url"
	"os"
	"path"
	"strings"

	"gitlab.com/gitlab-org/cli/internal/prompt"

	"gitlab.com/gitlab-org/cli/internal/glrepo"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/internal/git"
	"gitlab.com/gitlab-org/cli/internal/run"
)

var currentUser = func(client *gitlab.Client) (*gitlab.User, error) {
	u, _, err := client.Users.CurrentUser()
	if err != nil {
		return nil, err
	}
	return u, nil
}

var createProject = func(client *gitlab.Client, opts *gitlab.CreateProjectOptions) (*gitlab.Project, error) {
	project, _, err := client.Projects.CreateProject(opts)
	if err != nil {
		return nil, err
	}
	return project, nil
}

func NewCmdCreate(f cmdutils.Factory) *cobra.Command {
	projectCreateCmd := &cobra.Command{
		Use:   "create [path] [flags]",
		Short: `Create a new GitLab project/repository.`,
		Long:  ``,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreateProject(cmd, args, f)
		},
		Example: heredoc.Doc(`
			# Create a repository under your account using the current directory name.
			$ glab repo create

			# Create a repository under a group using the current directory name.
			$ glab repo create --group glab-cli

			# Create a repository with a specific name.
			$ glab repo create my-project

			# Create a repository for a group.
			$ glab repo create glab-cli/my-project
	  `),
	}

	projectCreateCmd.Flags().StringP("name", "n", "", "Name of the new project.")
	projectCreateCmd.Flags().StringP("group", "g", "", "Namespace or group for the new project. Defaults to the current user's namespace.")
	projectCreateCmd.Flags().StringP("description", "d", "", "Description of the new project.")
	projectCreateCmd.Flags().String("defaultBranch", "", "Default branch of the project. Defaults to `master` if not provided.")
	projectCreateCmd.Flags().String("remoteName", "origin", "Remote name for the Git repository you're in. Defaults to `origin` if not provided.")
	projectCreateCmd.Flags().StringArrayP("tag", "t", []string{}, "The list of tags for the project.")
	projectCreateCmd.Flags().Bool("internal", false, "Make project internal: visible to any authenticated user. Default.")
	projectCreateCmd.Flags().BoolP("private", "p", false, "Make project private: visible only to project members.")
	projectCreateCmd.Flags().BoolP("public", "P", false, "Make project public: visible without any authentication.")
	projectCreateCmd.Flags().Bool("readme", false, "Initialize project with `README.md`.")
	projectCreateCmd.Flags().BoolP("skipGitInit", "s", false, "Skip run 'git init'.")

	return projectCreateCmd
}

func runCreateProject(cmd *cobra.Command, args []string, f cmdutils.Factory) error {
	var (
		projectPath string
		visiblity   gitlab.VisibilityValue
		err         error
		isPath      bool
		namespaceID int
		namespace   string
	)
	c := f.IO().Color()

	defaultBranch, err := cmd.Flags().GetString("defaultBranch")
	if err != nil {
		return err
	}
	remoteName, err := cmd.Flags().GetString("remoteName")
	if err != nil {
		return err
	}
	skipGitInit, _ := cmd.Flags().GetBool("skipGitInit")
	if !skipGitInit && f.IO().PromptEnabled() {
		err = initGit(defaultBranch)
		if err != nil {
			return err
		}
	}

	var gitlabClient *gitlab.Client
	if len(args) == 1 {
		var host string
		host, namespace, projectPath = projectPathFromArgs(args, f.DefaultHostname())
		client, err := f.ApiClient(host, f.Config())
		if err != nil {
			return err
		}
		gitlabClient = client.Lab()

		user, err := currentUser(gitlabClient)
		if err != nil {
			return err
		}
		if user.Username == namespace {
			namespace = ""
		}
	} else {
		projectPath, err = git.ToplevelDir()
		if err != nil {
			return err
		}
		projectPath = path.Base(projectPath)
		isPath = true

		c, err := f.ApiClient(f.DefaultHostname(), f.Config())
		if err != nil {
			return err
		}
		gitlabClient = c.Lab()
	}

	group, err := cmd.Flags().GetString("group")
	if err != nil {
		return fmt.Errorf("could not parse group flag: %v", err)
	}
	if group != "" {
		namespace = group
	}

	if namespace != "" {
		group, _, err := gitlabClient.Groups.GetGroup(namespace, &gitlab.GetGroupOptions{})
		if err != nil {
			return fmt.Errorf("could not find group or namespace %s: %v", namespace, err)
		}
		namespaceID = group.ID
	}

	name, _ := cmd.Flags().GetString("name")

	if projectPath == "" && name == "" {
		fmt.Println("ERROR: path or name required to create a project.")
		return cmd.Usage()
	} else if name == "" {
		name = projectPath
	}

	description, _ := cmd.Flags().GetString("description")

	if internal, _ := cmd.Flags().GetBool("internal"); internal {
		visiblity = gitlab.InternalVisibility
	} else if private, _ := cmd.Flags().GetBool("private"); private {
		visiblity = gitlab.PrivateVisibility
	} else if public, _ := cmd.Flags().GetBool("public"); public {
		visiblity = gitlab.PublicVisibility
	}

	tags, _ := cmd.Flags().GetStringArray("tag")
	readme, _ := cmd.Flags().GetBool("readme")

	opts := &gitlab.CreateProjectOptions{
		Name:                 gitlab.Ptr(name),
		Path:                 gitlab.Ptr(projectPath),
		Description:          gitlab.Ptr(description),
		DefaultBranch:        gitlab.Ptr(defaultBranch),
		TagList:              &tags,
		InitializeWithReadme: gitlab.Ptr(readme),
	}

	if visiblity != "" {
		opts.Visibility = &visiblity
	}

	if namespaceID != 0 {
		opts.NamespaceID = &namespaceID
	}

	project, err := createProject(gitlabClient, opts)

	greenCheck := c.Green("✓")

	if err == nil {
		fmt.Fprintf(f.IO().StdOut, "%s Created repository %s on GitLab: %s\n", greenCheck, project.NameWithNamespace, project.WebURL)
		if isPath {
			cfg := f.Config()
			webURL, _ := url.Parse(project.WebURL)
			protocol, _ := cfg.Get(webURL.Host, "git_protocol")

			remote := glrepo.RemoteURL(project, protocol)
			_, err = git.AddRemote(remoteName, remote)
			if err != nil {
				return err
			}
			fmt.Fprintf(f.IO().StdOut, "%s Added remote %s\n", greenCheck, remote)

		} else if f.IO().PromptEnabled() {
			var doSetup bool
			err := prompt.Confirm(&doSetup, fmt.Sprintf("Create a local project directory for %s?", project.NameWithNamespace), true)
			if err != nil {
				return err
			}

			if doSetup {
				projectPath := project.Path
				err = initialiseRepo(projectPath, project.SSHURLToRepo)
				if err != nil {
					return err
				}
				fmt.Fprintf(f.IO().StdOut, "%s Initialized repository in './%s/'\n", greenCheck, projectPath)
			}
		}
	} else {
		return fmt.Errorf("error creating project: %v", err)
	}
	return err
}

func initGit(defaultBranch string) error {
	gitDir := path.Join(config.GitDir(false)...)
	if stat, err := os.Stat(gitDir); err == nil && stat.Mode().IsDir() {
		return nil
	}
	var doInit bool
	err := prompt.Confirm(&doInit, "Directory not Git initialized. Run `git init`?", true)
	if err != nil || !doInit {
		return err
	}

	gitInit := git.GitCommand("init")
	gitInit.Stdout = os.Stdout
	gitInit.Stderr = os.Stderr
	err = run.PrepareCmd(gitInit).Run()
	if err != nil {
		return err
	}

	if defaultBranch == "" {
		return nil
	}

	gitBranch := git.GitCommand("checkout", "-b", defaultBranch)
	gitBranch.Stdout = os.Stdout
	gitBranch.Stdin = os.Stdin
	return run.PrepareCmd(gitBranch).Run()
}

func initialiseRepo(projectPath, remoteURL string) error {
	gitInit := git.GitCommand("init", projectPath)
	gitInit.Stdout = os.Stdout
	gitInit.Stderr = os.Stderr
	err := run.PrepareCmd(gitInit).Run()
	if err != nil {
		return err
	}
	gitRemoteAdd := git.GitCommand("-C", projectPath, "remote", "add", "origin", remoteURL)
	gitRemoteAdd.Stdout = os.Stdout
	gitRemoteAdd.Stderr = os.Stderr
	err = run.PrepareCmd(gitRemoteAdd).Run()
	if err != nil {
		return err
	}
	return nil
}

func projectPathFromArgs(args []string, defaultHostname string) (string, string, string) {
	// sanitize input by removing trailing "/"
	project := strings.TrimSuffix(args[0], "/")

	var host, namespace string
	if strings.Contains(project, "/") {
		pp, _ := glrepo.FromFullName(project, defaultHostname)
		host = pp.RepoHost()
		project = pp.RepoName()
		namespace = pp.RepoNamespace()
	}
	return host, namespace, project
}
