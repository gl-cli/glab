package fork

import (
	"errors"
	"fmt"
	"net/url"

	"gitlab.com/gitlab-org/cli/internal/api"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/internal/git"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/internal/iostreams"
	"gitlab.com/gitlab-org/cli/internal/prompt"
	"gitlab.com/gitlab-org/cli/internal/run"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

type options struct {
	clone     bool
	addRemote bool
	repo      string
	name      string
	path      string

	cloneSet     bool
	addRemoteSet bool
	isTerminal   bool

	// whether the user specified the repo to clone
	// if false current git repo will be cloned
	currentDirIsParent bool

	repoToFork      glrepo.Interface
	io              *iostreams.IOStreams
	baseRepo        func() (glrepo.Interface, error)
	remotes         func() (glrepo.Remotes, error)
	config          func() config.Config
	apiClient       func(repoHost string, cfg config.Config) (*api.Client, error)
	defaultHostname string
}

func NewCmdFork(f cmdutils.Factory) *cobra.Command {
	opts := &options{
		io:                 f.IO(),
		baseRepo:           f.BaseRepo,
		remotes:            f.Remotes,
		config:             f.Config,
		apiClient:          f.ApiClient,
		defaultHostname:    f.DefaultHostname(),
		currentDirIsParent: true,
	}
	forkCmd := &cobra.Command{
		Use:   "fork <repo>",
		Short: "Fork a GitLab repository.",
		Example: heredoc.Doc(`
			$ glab repo fork
			$ glab repo fork namespace/repo
			$ glab repo fork namespace/repo --clone
		`),
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.complete(cmd, args)

			return opts.run()
		},
	}

	forkCmd.Flags().StringVarP(&opts.name, "name", "n", "", "The name assigned to the new project after forking.")
	forkCmd.Flags().StringVarP(&opts.path, "path", "p", "", "The path assigned to the new project after forking.")
	forkCmd.Flags().BoolVarP(&opts.clone, "clone", "c", false, "Clone the fork. Options: true, false.")
	forkCmd.Flags().BoolVar(&opts.addRemote, "remote", false, "Add a remote for the fork. Options: true, false.")

	return forkCmd
}

func (o *options) complete(cmd *cobra.Command, args []string) {
	if len(args) > 0 {
		o.repo = args[0]
		o.currentDirIsParent = false
	}

	o.cloneSet = cmd.Flags().Changed("clone")
	o.addRemoteSet = cmd.Flags().Changed("remote")
	o.isTerminal = o.io.IsaTTY && o.io.IsErrTTY && o.io.IsInTTY
}

func (o *options) run() error {
	var err error

	c := o.io.Color()
	if o.repo != "" {
		if git.IsValidURL(o.repo) {
			u, err := url.Parse(o.repo)
			if err != nil {
				return fmt.Errorf("invalid argument: %w", err)
			}
			o.repoToFork, err = glrepo.FromURL(u, o.defaultHostname)
			if err != nil {
				return fmt.Errorf("invalid argument: %w", err)
			}
		} else {
			o.repoToFork, err = glrepo.FromFullName(o.repo, o.defaultHostname)
			if err != nil {
				return fmt.Errorf("argument error: %w", err)
			}
		}
	} else {
		o.repoToFork, err = o.baseRepo()
		if err != nil {
			return fmt.Errorf("unable to determine source repository: %w", err)
		}
	}

	cfg := o.config()

	apiClient, err := o.apiClient(o.repoToFork.RepoHost(), cfg)
	if err != nil {
		return err
	}
	labClient := apiClient.Lab()

	if o.isTerminal {
		fmt.Fprintf(o.io.StdErr, "- Forking %s\n", c.Bold(o.repoToFork.FullName()))
	}

	forkOpts := &gitlab.ForkProjectOptions{}
	if o.name != "" {
		forkOpts.Name = gitlab.Ptr(o.name)
	}
	if o.path != "" {
		forkOpts.Path = gitlab.Ptr(o.path)
	}

	forkedProject, _, err := labClient.Projects.ForkProject(o.repoToFork.FullName(), forkOpts)
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
			forkedProject, err = api.GetProject(labClient, forkedProject.ID)
			if err != nil {
				fmt.Fprintf(o.io.StdErr, "error checking fork status: %q", err.Error())
				if retries == maximumRetries {
					break loop
				}
				fmt.Fprintln(o.io.StdErr, "- Retrying...")
				retries++
				continue
			}
		}
		skipFirstCheck = false
		// check import status of Fork
		// Import status should be one of {none, failed, scheduled, started, finished}
		// https://docs.gitlab.com/api/project_import_export/#import-status
		switch forkedProject.ImportStatus {
		case "none": // no import initiated
			break loop
		case importStatus:
			continue
		case "scheduled", "started": // import scheduled or started
			if importStatus != forkedProject.ImportStatus { // avoid printing the same message again
				fmt.Fprintln(o.io.StdErr, "- "+forkedProject.ImportStatus)
				importStatus = forkedProject.ImportStatus
			}
		case "finished": // import completed
			fmt.Fprintln(o.io.StdErr, "- "+forkedProject.ImportStatus)
			break loop
		case "failed": // import failed
			importError = errors.New(forkedProject.ImportError) // return the import error
			break loop
		default:
			break loop
		}
	}

	if importError != nil {
		fmt.Fprintf(o.io.StdErr, "%s: %q", c.Red("Fork failed"), importError.Error())
		return nil
	}

	fmt.Fprintf(o.io.StdErr, "%s Created fork %s.\n", c.GreenCheck(), forkedProject.PathWithNamespace)

	if (!o.isTerminal && o.currentDirIsParent && (!o.addRemote && o.addRemoteSet)) ||
		(!o.currentDirIsParent && (!o.clone && o.addRemoteSet)) {
		return nil
	}

	protocol, err := cfg.Get(o.repoToFork.RepoHost(), "git_protocol")
	if err != nil {
		return err
	}
	if o.currentDirIsParent {
		remotes, err := o.remotes()
		if err != nil {
			return err
		}

		if remote, err := remotes.FindByRepo(o.repoToFork.RepoOwner(), o.repoToFork.RepoName()); err == nil {

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
			if o.isTerminal {
				fmt.Fprintf(o.io.StdErr, "%s Using existing remote %s.\n", c.GreenCheck(), c.Bold(remote.Name))
			}
			return nil
		}

		remoteDesired := o.addRemote
		if !o.addRemoteSet {
			err = prompt.Confirm(&remoteDesired, "Would you like to add a remote for the fork?", true)
			if err != nil {
				return fmt.Errorf("failed to prompt: %w", err)
			}
		}
		if remoteDesired {
			remoteName := "origin"

			remotes, err := o.remotes()
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
				if o.isTerminal {
					fmt.Fprintf(o.io.StdErr, "%s Renamed %s remote to %s\n", c.GreenCheck(), c.Bold(remoteName), c.Bold(renameTarget))
				}
			}

			forkedRepoCloneURL := glrepo.RemoteURL(forkedProject, protocol)

			_, err = git.AddRemote(remoteName, forkedRepoCloneURL)
			if err != nil {
				return fmt.Errorf("failed to add remote: %w", err)
			}

			if o.isTerminal {
				fmt.Fprintf(o.io.StdErr, "%s Added remote %s.\n", c.GreenCheck(), c.Bold(remoteName))
			}
		}
	} else {
		cloneDesired := o.clone
		if !cloneDesired {
			// If clone is explicitly set to false exit
			if o.cloneSet {
				return nil
			}

			err = prompt.Confirm(&cloneDesired, "Would you like to clone the fork?", true)
			if err != nil {
				return fmt.Errorf("failed to prompt: %w", err)
			}
		}
		if cloneDesired {
			repoToFork, err := api.GetProject(labClient, o.repoToFork.FullName())
			if err != nil {
				return err
			}

			forkedRepoURL := glrepo.RemoteURL(forkedProject, protocol)

			cloneDir, err := git.RunClone(forkedRepoURL, "", []string{})
			if err != nil {
				return fmt.Errorf("failed to clone fork: %w", err)
			}

			upstreamURL := glrepo.RemoteURL(repoToFork, protocol)

			err = git.AddUpstreamRemote(upstreamURL, cloneDir)
			if err != nil {
				return err
			}

			if o.isTerminal {
				fmt.Fprintf(o.io.StdErr, "%s Cloned fork.\n", c.GreenCheck())
			}
		}
	}

	return nil
}
