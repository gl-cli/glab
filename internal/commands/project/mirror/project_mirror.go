package mirror

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/cli/internal/api"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/internal/iostreams"
)

type options struct {
	url                   string
	direction             string
	enabled               bool
	protectedBranchesOnly bool
	allowDivergence       bool
	projectID             int

	io               *iostreams.IOStreams
	baseRepo         glrepo.Interface
	apiClient        func(repoHost string, cfg config.Config) (*api.Client, error)
	gitlabClientFunc func() (*gitlab.Client, error)
	httpClient       *gitlab.Client
	config           func() config.Config
	baseRepoFactory  func() (glrepo.Interface, error)
	defaultHostname  string
}

func NewCmdMirror(f cmdutils.Factory) *cobra.Command {
	opts := options{
		io:               f.IO(),
		apiClient:        f.ApiClient,
		gitlabClientFunc: f.HttpClient,
		config:           f.Config,
		defaultHostname:  f.DefaultHostname(),
	}

	projectMirrorCmd := &cobra.Command{
		Use:   "mirror [ID | URL | PATH] [flags]",
		Short: "Mirror a project or repository to the specified location, using pull or push methods.",
		Long:  ``,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := opts.complete(args); err != nil {
				return err
			}
			if err := opts.validate(); err != nil {
				return err
			}

			return opts.run()
		},
	}
	projectMirrorCmd.Flags().StringVar(&opts.url, "url", "", "The target URL to which the repository is mirrored.")
	projectMirrorCmd.Flags().StringVar(&opts.direction, "direction", "pull", "Mirror direction. Options: pull, push.")
	projectMirrorCmd.Flags().BoolVar(&opts.enabled, "enabled", true, "Determines if the mirror is enabled.")
	projectMirrorCmd.Flags().BoolVar(&opts.protectedBranchesOnly, "protected-branches-only", false, "Determines if only protected branches are mirrored.")
	projectMirrorCmd.Flags().BoolVar(&opts.allowDivergence, "allow-divergence", false, "Determines if divergent refs are skipped.")

	_ = projectMirrorCmd.MarkFlagRequired("url")
	_ = projectMirrorCmd.MarkFlagRequired("direction")

	return projectMirrorCmd
}

func (o *options) complete(args []string) error {
	if len(args) > 0 {
		baseRepo, err := glrepo.FromFullName(args[0], o.defaultHostname)
		if err != nil {
			return err
		}
		o.baseRepo = baseRepo

		o.gitlabClientFunc = func() (*gitlab.Client, error) {
			if o.httpClient != nil {
				return o.httpClient, nil
			}
			cfg := o.config()
			c, err := o.apiClient(o.baseRepo.RepoHost(), cfg)
			if err != nil {
				return nil, err
			}
			o.httpClient = c.Lab()
			return o.httpClient, nil
		}

	} else {
		baseRepo, err := o.baseRepoFactory()
		if err != nil {
			return err
		}
		o.baseRepo = baseRepo
	}

	o.url = strings.TrimSpace(o.url)

	httpClient, err := o.gitlabClientFunc()
	if err != nil {
		return err
	}
	o.httpClient = httpClient

	project, err := o.baseRepo.Project(o.httpClient)
	if err != nil {
		return err
	}
	o.projectID = project.ID

	return nil
}

func (o *options) validate() error {
	if o.direction != "pull" && o.direction != "push" {
		return cmdutils.WrapError(
			errors.New("invalid choice for --direction"),
			"the argument direction value should be 'pull' or 'push'.",
		)
	}

	if o.direction == "pull" && o.allowDivergence {
		fmt.Fprintf(
			o.io.StdOut,
			"[Warning] the 'allow-divergence' flag has no effect for pull mirroring, and is ignored.\n",
		)
	}

	return nil
}

func (o *options) run() error {
	if o.direction == "push" {
		return o.createPushMirror()
	} else {
		return o.createPullMirror()
	}
}

func (o *options) createPushMirror() error {
	pm, _, err := o.httpClient.ProjectMirrors.AddProjectMirror(o.projectID, &gitlab.AddProjectMirrorOptions{
		URL:                   gitlab.Ptr(o.url),
		Enabled:               gitlab.Ptr(o.enabled),
		OnlyProtectedBranches: gitlab.Ptr(o.protectedBranchesOnly),
		KeepDivergentRefs:     gitlab.Ptr(o.allowDivergence),
	})
	if err != nil {
		return cmdutils.WrapError(err, "Failed to create push mirror.")
	}
	greenCheck := o.io.Color().Green("✓")
	fmt.Fprintf(
		o.io.StdOut,
		"%s Created push mirror for %s (%d) on GitLab at %s (%d).\n",
		greenCheck, pm.URL, pm.ID, o.baseRepo.FullName(), o.projectID,
	)
	return err
}

func (o *options) createPullMirror() error {
	_, _, err := o.httpClient.Projects.EditProject(o.projectID, &gitlab.EditProjectOptions{
		ImportURL:                   gitlab.Ptr(o.url),
		Mirror:                      gitlab.Ptr(o.enabled),
		OnlyMirrorProtectedBranches: gitlab.Ptr(o.protectedBranchesOnly),
	})
	if err != nil {
		return cmdutils.WrapError(err, "Failed to create pull mirror.")
	}
	greenCheck := o.io.Color().Green("✓")
	fmt.Fprintf(
		o.io.StdOut,
		"%s Created pull mirror for %s on GitLab at %s (%d).\n",
		greenCheck, o.url, o.baseRepo.FullName(), o.projectID,
	)
	return err
}
