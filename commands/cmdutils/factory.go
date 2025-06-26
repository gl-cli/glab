package cmdutils

import (
	"fmt"
	"strings"
	"sync"

	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/pkg/git"
	"gitlab.com/gitlab-org/cli/pkg/glinstance"
	"gitlab.com/gitlab-org/cli/pkg/iostreams"
)

// Factory is a way to obtain core tools for the commands.
// Safe for concurrent use.
type Factory interface {
	RepoOverride(repo string)
	ApiClient(repoHost string, cfg config.Config) (*api.Client, error)
	HttpClient() (*gitlab.Client, error)
	BaseRepo() (glrepo.Interface, error)
	Remotes() (glrepo.Remotes, error)
	Config() config.Config
	Branch() (string, error)
	IO() *iostreams.IOStreams
}

type DefaultFactory struct {
	io           *iostreams.IOStreams
	config       config.Config
	resolveRepos bool
	buildInfo    api.BuildInfo

	mu           sync.Mutex // protects the fields below
	repoOverride string
}

func NewFactory(io *iostreams.IOStreams, resolveRepos bool, cfg config.Config, buildInfo api.BuildInfo) *DefaultFactory {
	return &DefaultFactory{
		io:           io,
		config:       cfg,
		resolveRepos: resolveRepos,
		buildInfo:    buildInfo,
	}
}

func (f *DefaultFactory) RepoOverride(repo string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.repoOverride = repo
}

func (f *DefaultFactory) ApiClient(repoHost string, cfg config.Config) (*api.Client, error) {
	c, err := api.NewClientWithCfg(repoHost, cfg, false, f.buildInfo.UserAgent())
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (f *DefaultFactory) HttpClient() (*gitlab.Client, error) {
	cfg := f.Config()

	f.mu.Lock()
	override := f.repoOverride
	f.mu.Unlock()
	var repo glrepo.Interface
	if override != "" {
		var err error
		repo, err = glrepo.FromFullName(override)
		if err != nil {
			return nil, err // return the error if repo was overridden.
		}
	} else {
		remotes, err := f.Remotes()
		if err != nil {
			// use default hostname if remote resolver fails
			repo = glrepo.NewWithHost("", "", glinstance.OverridableDefault())
		} else {
			repo = remotes[0]
		}
	}

	c, err := api.NewClientWithCfg(repo.RepoHost(), cfg, false, f.buildInfo.UserAgent())
	if err != nil {
		return nil, err
	}

	return c.Lab(), nil
}

func (f *DefaultFactory) BaseRepo() (glrepo.Interface, error) {
	f.mu.Lock()
	override := f.repoOverride
	f.mu.Unlock()
	if override != "" {
		return glrepo.FromFullName(override)
	}
	remotes, err := f.Remotes()
	if err != nil {
		return nil, err
	}
	if !f.resolveRepos {
		return remotes[0], nil
	}
	cfg := f.Config()
	ac, err := api.NewClientWithCfg(remotes[0].RepoHost(), cfg, false, f.buildInfo.UserAgent())
	if err != nil {
		return nil, err
	}
	httpClient := ac.Lab()
	repoContext, err := glrepo.ResolveRemotesToRepos(remotes, httpClient, "")
	if err != nil {
		return nil, err
	}
	baseRepo, err := repoContext.BaseRepo(f.io.PromptEnabled())
	if err != nil {
		return nil, err
	}
	return baseRepo, nil
}

func (f *DefaultFactory) Remotes() (glrepo.Remotes, error) {
	hostOverride := ""
	if !strings.EqualFold(glinstance.Default(), glinstance.OverridableDefault()) {
		hostOverride = glinstance.OverridableDefault()
	}
	rr := &remoteResolver{
		readRemotes: git.Remotes,
		getConfig:   f.Config,
	}
	fn := rr.Resolver(hostOverride)
	return fn()
}

func (f *DefaultFactory) Config() config.Config {
	return f.config
}

func (f *DefaultFactory) Branch() (string, error) {
	currentBranch, err := git.CurrentBranch()
	if err != nil {
		return "", fmt.Errorf("could not determine current branch: %w", err)
	}
	return currentBranch, nil
}

func (f *DefaultFactory) IO() *iostreams.IOStreams {
	return f.io
}
