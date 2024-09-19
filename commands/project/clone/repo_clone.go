package clone

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"gitlab.com/gitlab-org/cli/pkg/iostreams"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/xanzy/go-gitlab"
	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/pkg/git"
	"gitlab.com/gitlab-org/cli/pkg/glinstance"
)

type CloneOptions struct {
	GroupName         string
	IncludeSubgroups  bool
	PreserveNamespace bool
	WithMREnabled     bool
	WithIssuesEnabled bool
	WithShared        bool
	Archived          bool
	ArchivedSet       bool
	Visibility        string
	Owned             bool
	GitFlags          []string
	Dir               string
	Host              string
	Protocol          string

	Page     int
	PerPage  int
	Paginate bool

	IO        *iostreams.IOStreams
	APIClient *api.Client
	Config    func() (config.Config, error)

	CurrentUser *gitlab.User
}

type ContextOpts struct {
	Project *gitlab.Project
	Repo    string
}

func NewCmdClone(f *cmdutils.Factory, runE func(*CloneOptions, *ContextOpts) error) *cobra.Command {
	opts := &CloneOptions{
		IO:     f.IO,
		Config: f.Config,
	}

	ctxOpts := &ContextOpts{}

	repoCloneCmd := &cobra.Command{
		Use:   "clone <repo> [flags] [<dir>] [-- [<gitflags>...]]",
		Short: `Clone a GitLab repository or project.`,
		Example: heredoc.Doc(`
	$ glab repo clone profclems/glab

	$ glab repo clone https://gitlab.com/profclems/glab

	# Clones repository into 'mydirectory'
	$ glab repo clone profclems/glab mydirectory

	# Clones repository 'glab' for current user
	$ glab repo clone glab

	# Finds the project by the ID provided and clones it
	$ glab repo clone 4356677

	# Clones all repos in a group
	$ glab repo clone -g everyonecancontribute --paginate

	# Clones all non-archived repos in a group
	$ glab repo clone -g everyonecancontribute --archived=false --paginate

	# Clones from a self-hosted instance
	$ GITLAB_HOST=salsa.debian.org glab repo clone myrepo
	`),
		Long: heredoc.Doc(`
		Clone supports these shorthand references:

		- repo
		- namespace/repo
		- org/group/repo
		- project ID
	`),
		RunE: func(cmd *cobra.Command, args []string) error {
			if nArgs := len(args); nArgs > 0 {
				ctxOpts.Repo = args[0]
				if nArgs > 1 && !opts.PreserveNamespace {
					opts.Dir = args[1]
				}
				opts.GitFlags = args[1:]
			}

			if ctxOpts.Repo == "" && opts.GroupName == "" {
				return &cmdutils.FlagError{Err: fmt.Errorf("Specify repository argument, or use the --group flag to specify a group to clone all repos from the group.")}
			}

			if runE != nil {
				return runE(opts, ctxOpts)
			}

			opts.Host = glinstance.OverridableDefault()
			opts.ArchivedSet = cmd.Flags().Changed("archived")

			cfg, err := opts.Config()
			if err != nil {
				return err
			}
			opts.APIClient, err = api.NewClientWithCfg(opts.Host, cfg, false)
			if err != nil {
				return err
			}

			opts.CurrentUser, err = api.CurrentUser(opts.APIClient.Lab())
			if err != nil {
				return err
			}

			opts.Protocol, _ = cfg.Get(opts.Host, "git_protocol")

			if opts.GroupName != "" {
				return groupClone(opts, ctxOpts)
			}

			return cloneRun(opts, ctxOpts)
		},
	}

	repoCloneCmd.Flags().StringVarP(&opts.GroupName, "group", "g", "", "Specify the group to clone repositories from.")
	repoCloneCmd.Flags().BoolVarP(&opts.PreserveNamespace, "preserve-namespace", "p", false, "Clone the repository in a subdirectory based on namespace.")
	repoCloneCmd.Flags().BoolVarP(&opts.Archived, "archived", "a", false, "Limit by archived status. Use with '-a=false' to exclude archived repositories. Used with the --group flag.")
	repoCloneCmd.Flags().BoolVarP(&opts.IncludeSubgroups, "include-subgroups", "G", true, "Include projects in subgroups of this group. Default is true. Used with the --group flag.")
	repoCloneCmd.Flags().BoolVarP(&opts.Owned, "mine", "m", false, "Limit by projects in the group owned by the current authenticated user. Used with the --group flag.")
	repoCloneCmd.Flags().StringVarP(&opts.Visibility, "visibility", "v", "", "Limit by visibility: public, internal, private. Used with the --group flag.")
	repoCloneCmd.Flags().BoolVarP(&opts.WithIssuesEnabled, "with-issues-enabled", "I", false, "Limit by projects with the issues feature enabled. Default is false. Used with the --group flag.")
	repoCloneCmd.Flags().BoolVarP(&opts.WithMREnabled, "with-mr-enabled", "M", false, "Limit by projects with the merge request feature enabled. Default is false. Used with the --group flag.")
	repoCloneCmd.Flags().BoolVarP(&opts.WithShared, "with-shared", "S", true, "Include projects shared to this group. Default is true. Used with the --group flag.")
	repoCloneCmd.Flags().BoolVarP(&opts.Paginate, "paginate", "", false, "Make additional HTTP requests to fetch all pages of projects before cloning. Respects --per-page.")
	repoCloneCmd.Flags().IntVarP(&opts.Page, "page", "", 1, "Page number.")
	repoCloneCmd.Flags().IntVarP(&opts.PerPage, "per-page", "", 30, "Number of items to list per page.")

	repoCloneCmd.Flags().SortFlags = false
	repoCloneCmd.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		if errors.Is(err, pflag.ErrHelp) {
			return err
		}
		return &cmdutils.FlagError{Err: fmt.Errorf("%w\nSeparate Git clone flags with '--'.", err)}
	})

	return repoCloneCmd
}

func listProjects(opts *CloneOptions, ListGroupProjectOpts *gitlab.ListGroupProjectsOptions) ([]*gitlab.Project, error) {
	var projects []*gitlab.Project
	hasRemaining := true

	for hasRemaining {
		currentPage, resp, err := api.ListGroupProjects(opts.APIClient.Lab(), opts.GroupName, ListGroupProjectOpts)
		if err != nil {
			return nil, err
		}
		if len(currentPage) == 0 {
			fmt.Fprintf(opts.IO.StdErr, "Group %q does not have any projects.\n", opts.GroupName)
			return nil, cmdutils.SilentError
		}

		projects = append(projects, currentPage...)

		ListGroupProjectOpts.Page = resp.NextPage
		hasRemaining = opts.Paginate && resp.CurrentPage != resp.TotalPages
	}

	return projects, nil
}

func groupClone(opts *CloneOptions, ctxOpts *ContextOpts) error {
	c := opts.IO.Color()
	ListGroupProjectOpts := &gitlab.ListGroupProjectsOptions{}
	if !opts.WithShared {
		ListGroupProjectOpts.WithShared = gitlab.Ptr(false)
	}
	if opts.WithMREnabled {
		ListGroupProjectOpts.WithMergeRequestsEnabled = gitlab.Ptr(true)
	}
	if opts.WithIssuesEnabled {
		ListGroupProjectOpts.WithIssuesEnabled = gitlab.Ptr(true)
	}
	if opts.Owned {
		ListGroupProjectOpts.Owned = gitlab.Ptr(true)
	}
	if opts.ArchivedSet {
		ListGroupProjectOpts.Archived = gitlab.Ptr(opts.Archived)
	}
	if opts.IncludeSubgroups {
		includeSubGroups := true
		ListGroupProjectOpts.IncludeSubGroups = &includeSubGroups
	}
	if opts.Visibility != "" {
		ListGroupProjectOpts.Visibility = gitlab.Ptr(gitlab.VisibilityValue(opts.Visibility))
	}

	ListGroupProjectOpts.PerPage = 100
	if opts.Paginate {
		ListGroupProjectOpts.PerPage = 30
	}
	if opts.PerPage != 0 {
		ListGroupProjectOpts.PerPage = opts.PerPage
	}
	if opts.Page != 0 {
		ListGroupProjectOpts.Page = opts.Page
	}

	projects, err := listProjects(opts, ListGroupProjectOpts)
	var finalOutput []string
	for _, project := range projects {
		ctxOpt := *ctxOpts
		ctxOpt.Project = project
		ctxOpt.Repo = project.PathWithNamespace
		err = cloneRun(opts, &ctxOpt)
		if err != nil {
			finalOutput = append(finalOutput, fmt.Sprintf("%s %s - Error: %q", c.RedCheck(), project.PathWithNamespace, err.Error()))
		} else {
			finalOutput = append(finalOutput, fmt.Sprintf("%s %s", c.GreenCheck(), project.PathWithNamespace))
		}
	}

	// Print error/success msgs in human-readable formats
	for _, out := range finalOutput {
		fmt.Fprintln(opts.IO.StdOut, out)
	}
	if err != nil { // if any error came up
		return cmdutils.SilentError
	}
	return nil
}

func cloneRun(opts *CloneOptions, ctxOpts *ContextOpts) (err error) {
	if !git.IsValidURL(ctxOpts.Repo) {
		// Assuming that repo is a project ID if it is an integer
		if _, err := strconv.ParseInt(ctxOpts.Repo, 10, 64); err != nil {
			// Assuming that "/" in the project name means its owned by an organisation
			if !strings.Contains(ctxOpts.Repo, "/") {
				ctxOpts.Repo = fmt.Sprintf("%s/%s", opts.CurrentUser.Username, ctxOpts.Repo)
			}
		}
		if ctxOpts.Project == nil {
			ctxOpts.Project, err = api.GetProject(opts.APIClient.Lab(), ctxOpts.Repo)
			if err != nil {
				return
			}
		}
		ctxOpts.Repo = glrepo.RemoteURL(ctxOpts.Project, opts.Protocol)
	} else if !strings.HasSuffix(ctxOpts.Repo, ".git") {
		ctxOpts.Repo += ".git"
	}
	// To preserve namespaces, we deep copy gitFlags for group clones
	var gitFlags []string
	if opts.PreserveNamespace {
		namespacedDir := ctxOpts.Project.PathWithNamespace
		opts.Dir = namespacedDir
		gitFlags = append([]string{namespacedDir}, opts.GitFlags...)
	} else {
		gitFlags = opts.GitFlags
	}
	_, err = git.RunClone(ctxOpts.Repo, gitFlags)
	if err != nil {
		return
	}
	// Cloned project was a fork belonging to the user; user is
	// treating fork's ssh/https url as origin. Add upstream as remote pointing
	// to forked repo's ssh/https url depending on the users preferred protocol
	if ctxOpts.Project != nil {
		if ctxOpts.Project.ForkedFromProject != nil && strings.Contains(ctxOpts.Project.PathWithNamespace, opts.CurrentUser.Username) {
			if opts.Dir == "" {
				opts.Dir = "./" + ctxOpts.Project.Path
			}
			fProject, err := api.GetProject(opts.APIClient.Lab(), ctxOpts.Project.ForkedFromProject.PathWithNamespace)
			if err != nil {
				return err
			}
			repoURL := glrepo.RemoteURL(fProject, opts.Protocol)
			err = git.AddUpstreamRemote(repoURL, opts.Dir)
			if err != nil {
				return err
			}
		}
	}
	return
}
