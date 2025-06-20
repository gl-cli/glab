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
	HttpClient() (*gitlab.Client, error)
	BaseRepo() (glrepo.Interface, error)
	Remotes() (glrepo.Remotes, error)
	Config() (config.Config, error)
	Branch() (string, error)
	IO() *iostreams.IOStreams
}

type DefaultFactory struct {
	io           *iostreams.IOStreams
	resolveRepos bool

	mu           sync.Mutex // protects the fields below
	repoOverride string
	cachedConfig config.Config
	configError  error
}

func NewFactory(io *iostreams.IOStreams, resolveRepos bool) *DefaultFactory {
	return &DefaultFactory{
		io:           io,
		resolveRepos: resolveRepos,
	}
}

func NewFactoryWithConfig(io *iostreams.IOStreams, resolveRepos bool, cfg config.Config) *DefaultFactory {
	return &DefaultFactory{
		io:           io,
		resolveRepos: resolveRepos,
		cachedConfig: cfg,
	}
}

func (f *DefaultFactory) RepoOverride(repo string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.repoOverride = repo
}

func (f *DefaultFactory) HttpClient() (*gitlab.Client, error) {
	cfg, err := f.Config()
	if err != nil {
		return nil, err
	}
	f.mu.Lock()
	override := f.repoOverride
	f.mu.Unlock()
	var repo glrepo.Interface
	if override != "" {
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
	return LabClientFunc(repo.RepoHost(), cfg, false)
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
	cfg, err := f.Config()
	if err != nil {
		return nil, err
	}
	httpClient, err := LabClientFunc(remotes[0].RepoHost(), cfg, false)
	if err != nil {
		return nil, err
	}
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

func (f *DefaultFactory) Config() (config.Config, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.cachedConfig != nil || f.configError != nil {
		return f.cachedConfig, f.configError
	}
	f.cachedConfig, f.configError = config.Init()
	return f.cachedConfig, f.configError
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

func LabClientFunc(repoHost string, cfg config.Config, isGraphQL bool) (*gitlab.Client, error) {
	c, err := api.NewClientWithCfg(repoHost, cfg, isGraphQL)
	if err != nil {
		return nil, err
	}
	return c.Lab(), nil
}
