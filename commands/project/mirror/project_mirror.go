package mirror

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/xanzy/go-gitlab"
	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/pkg/iostreams"
)

type MirrorOptions struct {
	URL                   string
	Direction             string
	Enabled               bool
	ProtectedBranchesOnly bool
	AllowDivergence       bool
	ProjectID             int

	IO         *iostreams.IOStreams
	BaseRepo   glrepo.Interface
	APIClient  func() (*gitlab.Client, error)
	httpClient *gitlab.Client
}

func NewCmdMirror(f *cmdutils.Factory) *cobra.Command {
	opts := MirrorOptions{
		IO: f.IO,
	}

	projectMirrorCmd := &cobra.Command{
		Use:   "mirror [ID | URL | PATH] [flags]",
		Short: "Mirror a project or repository to the specified location, using pull or push methods.",
		Long:  ``,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error

			opts.APIClient = f.HttpClient

			if len(args) > 0 {
				opts.BaseRepo, err = glrepo.FromFullName(args[0])
				if err != nil {
					return err
				}

				opts.APIClient = func() (*gitlab.Client, error) {
					if opts.httpClient != nil {
						return opts.httpClient, nil
					}
					cfg, err := f.Config()
					if err != nil {
						return nil, err
					}
					c, err := api.NewClientWithCfg(opts.BaseRepo.RepoHost(), cfg, false)
					if err != nil {
						return nil, err
					}
					opts.httpClient = c.Lab()
					return opts.httpClient, nil
				}

			} else {
				opts.BaseRepo, err = f.BaseRepo()
				if err != nil {
					return err
				}
			}

			if opts.Direction != "pull" && opts.Direction != "push" {
				return cmdutils.WrapError(
					errors.New("invalid choice for --direction"),
					"the argument direction value should be 'pull' or 'push'.",
				)
			}

			if opts.Direction == "pull" && opts.AllowDivergence {
				fmt.Fprintf(
					f.IO.StdOut,
					"[Warning] the 'allow-divergence' flag has no effect for pull mirroring, and is ignored.\n",
				)
			}

			opts.URL = strings.TrimSpace(opts.URL)

			opts.httpClient, err = opts.APIClient()
			if err != nil {
				return err
			}

			project, err := opts.BaseRepo.Project(opts.httpClient)
			if err != nil {
				return err
			}
			opts.ProjectID = project.ID
			return runProjectMirror(&opts)
		},
	}
	projectMirrorCmd.Flags().StringVar(&opts.URL, "url", "", "The target URL to which the repository is mirrored.")
	projectMirrorCmd.Flags().StringVar(&opts.Direction, "direction", "pull", "Mirror direction. Options: pull, push.")
	projectMirrorCmd.Flags().BoolVar(&opts.Enabled, "enabled", true, "Determines if the mirror is enabled.")
	projectMirrorCmd.Flags().BoolVar(&opts.ProtectedBranchesOnly, "protected-branches-only", false, "Determines if only protected branches are mirrored.")
	projectMirrorCmd.Flags().BoolVar(&opts.AllowDivergence, "allow-divergence", false, "Determines if divergent refs are skipped.")

	_ = projectMirrorCmd.MarkFlagRequired("url")
	_ = projectMirrorCmd.MarkFlagRequired("direction")

	return projectMirrorCmd
}

func runProjectMirror(opts *MirrorOptions) error {
	if opts.Direction == "push" {
		return createPushMirror(opts)
	} else {
		return createPullMirror(opts)
	}
}

func createPushMirror(opts *MirrorOptions) error {
	var pm *gitlab.ProjectMirror
	var err error
	pushOptions := api.CreatePushMirrorOptions{
		Url:                   opts.URL,
		Enabled:               opts.Enabled,
		OnlyProtectedBranches: opts.ProtectedBranchesOnly,
		KeepDivergentRefs:     opts.AllowDivergence,
	}
	pm, err = api.CreatePushMirror(
		opts.httpClient,
		opts.ProjectID,
		&pushOptions,
	)
	if err != nil {
		return cmdutils.WrapError(err, "Failed to create push mirror.")
	}
	greenCheck := opts.IO.Color().Green("✓")
	fmt.Fprintf(
		opts.IO.StdOut,
		"%s Created push mirror for %s (%d) on GitLab at %s (%d).\n",
		greenCheck, pm.URL, pm.ID, opts.BaseRepo.FullName(), opts.ProjectID,
	)
	return err
}

func createPullMirror(opts *MirrorOptions) error {
	pullOptions := api.CreatePullMirrorOptions{
		Url:                   opts.URL,
		Enabled:               opts.Enabled,
		OnlyProtectedBranches: opts.ProtectedBranchesOnly,
	}
	err := api.CreatePullMirror(
		opts.httpClient,
		opts.ProjectID,
		&pullOptions,
	)
	if err != nil {
		return cmdutils.WrapError(err, "Failed to create pull mirror.")
	}
	greenCheck := opts.IO.Color().Green("✓")
	fmt.Fprintf(
		opts.IO.StdOut,
		"%s Created pull mirror for %s on GitLab at %s (%d).\n",
		greenCheck, opts.URL, opts.BaseRepo.FullName(), opts.ProjectID,
	)
	return err
}
