package fork

import (
	"errors"
	"fmt"
	"net/url"

	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/internal/run"
	"gitlab.com/gitlab-org/cli/pkg/git"
	"gitlab.com/gitlab-org/cli/pkg/iostreams"
	"gitlab.com/gitlab-org/cli/pkg/prompt"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	"github.com/xanzy/go-gitlab"
)

type ForkOptions struct {
	Clone     bool
	AddRemote bool
	Repo      string
	Name      string
	Path      string

	CloneSet     bool
	AddRemoteSet bool
	IsTerminal   bool

	// whether the user specified the repo to clone
	// if false current git repo will be cloned
	CurrentDirIsParent bool

	RepoToFork  glrepo.Interface
	IO          *iostreams.IOStreams
	LabClient   *gitlab.Client
	CurrentUser *gitlab.User
	BaseRepo    func() (glrepo.Interface, error)
	Remotes     func() (glrepo.Remotes, error)
	Config      func() (config.Config, error)
}

func NewCmdFork(f *cmdutils.Factory, runE func(*cmdutils.Factory) error) *cobra.Command {
	opts := &ForkOptions{
		IO:                 f.IO,
		BaseRepo:           f.BaseRepo,
		Remotes:            f.Remotes,
		Config:             f.Config,
		CurrentDirIsParent: true,
	}
	forkCmd := &cobra.Command{
		Use:   "fork <repo>",
		Short: "Fork a GitLab repository.",
		Example: heredoc.Doc(`
			glab repo fork
			glab repo fork namespace/repo
			glab repo fork namespace/repo --clone
		`),
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			if len(args) > 0 {
				opts.Repo = args[0]
				opts.CurrentDirIsParent = false
			}

			opts.CloneSet = cmd.Flags().Changed("clone")
			opts.AddRemoteSet = cmd.Flags().Changed("remote")
			opts.IsTerminal = opts.IO.IsaTTY && opts.IO.IsErrTTY && opts.IO.IsInTTY

			if runE != nil {
				return runE(f)
			}

			opts.LabClient, err = f.HttpClient()
			if err != nil {
				return err
			}
			opts.CurrentUser, err = api.CurrentUser(opts.LabClient)
			if err != nil {
				return err
			}
			return forkRun(opts)
		},
	}

	forkCmd.Flags().StringVarP(&opts.Name, "name", "n", "", "The name assigned to the new project after forking.")
	forkCmd.Flags().StringVarP(&opts.Path, "path", "p", "", "The path assigned to the new project after forking.")
	forkCmd.Flags().BoolVarP(&opts.Clone, "clone", "c", false, "Clone the fork. Options: true, false.")
	forkCmd.Flags().BoolVar(&opts.AddRemote, "remote", false, "Add a remote for the fork. Options: true, false.")

	return forkCmd
}

func forkRun(opts *ForkOptions) error {
	var err error
	c := opts.IO.Color()
	if opts.Repo != "" {
		if git.IsValidURL(opts.Repo) {
			u, err := url.Parse(opts.Repo)
			if err != nil {
				return fmt.Errorf("invalid argument: %w", err)
			}
			opts.RepoToFork, err = glrepo.FromURL(u)
			if err != nil {
				return fmt.Errorf("invalid argument: %w", err)
			}
		} else {
			opts.RepoToFork, err = glrepo.FromFullName(opts.Repo)
			if err != nil {
				return fmt.Errorf("argument error: %w", err)
			}
		}
	} else {
		opts.RepoToFork, err = opts.BaseRepo()
		if err != nil {
			return fmt.Errorf("unable to determine source repository: %w", err)
		}
	}

	cfg, err := opts.Config()
	if err != nil {
		return err
	}

	apiClient, err := api.NewClientWithCfg(opts.RepoToFork.RepoHost(), cfg, false)
	if err != nil {
		return err
	}
	opts.LabClient = apiClient.LabClient

	if opts.IsTerminal {
		fmt.Fprintf(opts.IO.StdErr, "- Forking %s\n", c.Bold(opts.RepoToFork.FullName()))
	}

	forkOpts := &gitlab.ForkProjectOptions{}
	if opts.Name != "" {
		forkOpts.Name = gitlab.Ptr(opts.Name)
	}
	if opts.Path != "" {
		forkOpts.Path = gitlab.Ptr(opts.Path)
	}

	forkedProject, err := api.ForkProject(opts.LabClient, opts.RepoToFork.FullName(), forkOpts)
	if err != nil {
		return err
	}
	// The forking operation for a project is asynchronous and is completed in a background job.
	// The request returns immediately. To determine whether the fork of the project has completed,
	// we query the import_status for the new project.
	importStatus := ""
	var importError error
	maximumRetries := 3
	retries := 0
	skipFirstCheck := true
loop:
	for {
		if !skipFirstCheck {
			// get the forked project
			forkedProject, err = api.GetProject(opts.LabClient, forkedProject.ID)
			if err != nil {
				fmt.Fprintf(opts.IO.StdErr, "error checking fork status: %q", err.Error())
				if retries == maximumRetries {
					break loop
				}
				fmt.Fprintln(opts.IO.StdErr, "- Retrying...")
				retries++
				continue
			}
		}
		skipFirstCheck = false
		// check import status of Fork
		// Import status should be one of {none, failed, scheduled, started, finished}
		// https://docs.gitlab.com/ee/api/project_import_export.html#import-status
		switch forkedProject.ImportStatus {
		case "none": // no import initiated
			break loop
		case importStatus:
			continue
		case "scheduled", "started": // import scheduled or started
			if importStatus != forkedProject.ImportStatus { // avoid printing the same message again
				fmt.Fprintln(opts.IO.StdErr, "- "+forkedProject.ImportStatus)
				importStatus = forkedProject.ImportStatus
			}
		case "finished": // import completed
			fmt.Fprintln(opts.IO.StdErr, "- "+forkedProject.ImportStatus)
			break loop
		case "failed": // import failed
			importError = errors.New(forkedProject.ImportError) // return the import error
			break loop
		default:
			break loop
		}
	}

	if importError != nil {
		fmt.Fprintf(opts.IO.StdErr, "%s: %q", c.Red("Fork failed"), importError.Error())
		return nil
	}

	fmt.Fprintf(opts.IO.StdErr, "%s Created fork %s.\n", c.GreenCheck(), forkedProject.PathWithNamespace)

	if (!opts.IsTerminal && opts.CurrentDirIsParent && (!opts.AddRemote && opts.AddRemoteSet)) ||
		(!opts.CurrentDirIsParent && (!opts.Clone && opts.AddRemoteSet)) {
		return nil
	}

	protocol, err := cfg.Get(opts.RepoToFork.RepoHost(), "git_protocol")
	if err != nil {
		return err
	}
	if opts.CurrentDirIsParent {
		remotes, err := opts.Remotes()
		if err != nil {
			return err
		}

		if remote, err := remotes.FindByRepo(opts.RepoToFork.RepoOwner(), opts.RepoToFork.RepoName()); err == nil {

			scheme := ""
			if remote.FetchURL != nil {
				scheme = remote.FetchURL.Scheme
			}
			if remote.PushURL != nil {
				scheme = remote.PushURL.Scheme
			}
			if scheme != "" {
				protocol = scheme
			}
		}

		if remote, err := remotes.FindByRepo(forkedProject.Namespace.FullPath, forkedProject.Path); err == nil {
			if opts.IsTerminal {
				fmt.Fprintf(opts.IO.StdErr, "%s Using existing remote %s.\n", c.GreenCheck(), c.Bold(remote.Name))
			}
			return nil
		}

		remoteDesired := opts.AddRemote
		if !opts.AddRemoteSet {
			err = prompt.Confirm(&remoteDesired, "Would you like to add a remote for the fork?", true)
			if err != nil {
				return fmt.Errorf("failed to prompt: %w", err)
			}
		}
		if remoteDesired {
			remoteName := "origin"

			remotes, err := opts.Remotes()
			if err != nil {
				return err
			}
			if _, err := remotes.FindByName(remoteName); err == nil {
				renameTarget := "upstream"
				renameCmd := git.GitCommand("remote", "rename", remoteName, renameTarget)
				err = run.PrepareCmd(renameCmd).Run()
				if err != nil {
					return err
				}
				if opts.IsTerminal {
					fmt.Fprintf(opts.IO.StdErr, "%s Renamed %s remote to %s\n", c.GreenCheck(), c.Bold(remoteName), c.Bold(renameTarget))
				}
			}

			forkedRepoCloneURL := glrepo.RemoteURL(forkedProject, protocol)

			_, err = git.AddRemote(remoteName, forkedRepoCloneURL)
			if err != nil {
				return fmt.Errorf("failed to add remote: %w", err)
			}

			if opts.IsTerminal {
				fmt.Fprintf(opts.IO.StdErr, "%s Added remote %s.\n", c.GreenCheck(), c.Bold(remoteName))
			}
		}
	} else {
		cloneDesired := opts.Clone
		if !cloneDesired {
			// If clone is explicitly set to false exit
			if opts.CloneSet {
				return nil
			}

			err = prompt.Confirm(&cloneDesired, "Would you like to clone the fork?", true)
			if err != nil {
				return fmt.Errorf("failed to prompt: %w", err)
			}
		}
		if cloneDesired {
			repoToFork, err := api.GetProject(opts.LabClient, opts.RepoToFork.FullName())
			if err != nil {
				return err
			}

			forkedRepoURL := glrepo.RemoteURL(forkedProject, protocol)

			cloneDir, err := git.RunClone(forkedRepoURL, []string{})
			if err != nil {
				return fmt.Errorf("failed to clone fork: %w", err)
			}

			upstreamURL := glrepo.RemoteURL(repoToFork, protocol)

			err = git.AddUpstreamRemote(upstreamURL, cloneDir)
			if err != nil {
				return err
			}

			if opts.IsTerminal {
				fmt.Fprintf(opts.IO.StdErr, "%s Cloned fork.\n", c.GreenCheck())
			}
		}
	}

	return nil
}
