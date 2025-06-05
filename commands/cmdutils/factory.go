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

var (
	CachedConfig config.Config
	ConfigError  error
)

type Factory struct {
	HttpClient func() (*gitlab.Client, error)
	BaseRepo   func() (glrepo.Interface, error)
	Remotes    func() (glrepo.Remotes, error)
	Config     func() (config.Config, error)
	Branch     func() (string, error)
	IO         *iostreams.IOStreams

	clientOnce sync.Once
	client     *gitlab.Client
	clientErr  error
}

// MIT License
//
// Copyright (c) 2019 GitHub Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

func (f *Factory) RepoOverride(repo string) error {
	f.BaseRepo = func() (glrepo.Interface, error) {
		return glrepo.FromFullName(repo)
	}
	newRepo, err := f.BaseRepo()
	if err != nil {
		return err
	}
	// Initialise new http client for new repo host
	cfg, err := f.Config()
	if err != nil {
		return err
	}

	f.client = nil
	f.clientErr = nil
	f.clientOnce = sync.Once{}

	f.HttpClient = func() (*gitlab.Client, error) {
		return LabClientFunc(newRepo.RepoHost(), cfg, false)
	}
	return nil
}

func LabClientFunc(repoHost string, cfg config.Config, isGraphQL bool) (*gitlab.Client, error) {
	c, err := api.NewClientWithCfg(repoHost, cfg, isGraphQL)
	if err != nil {
		return nil, err
	}
	return c.Lab(), nil
}

func remotesFunc() (glrepo.Remotes, error) {
	hostOverride := ""
	if !strings.EqualFold(glinstance.Default(), glinstance.OverridableDefault()) {
		hostOverride = glinstance.OverridableDefault()
	}
	rr := &remoteResolver{
		readRemotes: git.Remotes,
		getConfig:   configFunc,
	}
	fn := rr.Resolver(hostOverride)
	return fn()
}

// FIXME remove usage of global variables
func configFunc() (config.Config, error) {
	if CachedConfig != nil || ConfigError != nil {
		return CachedConfig, ConfigError
	}
	CachedConfig, ConfigError = config.Init()
	return CachedConfig, ConfigError
}

func baseRepoFunc() (glrepo.Interface, error) {
	remotes, err := remotesFunc()
	if err != nil {
		return nil, err
	}
	return remotes[0], nil
}

// OverrideAPIProtocol sets api protocol for host to initialize http client
func OverrideAPIProtocol(cfg config.Config, repo glrepo.Interface) {
	protocol, _ := cfg.Get(repo.RepoHost(), "api_protocol")
	api.SetProtocol(protocol)
}

func httpClientFunc() (*gitlab.Client, error) {
	cfg, err := configFunc()
	if err != nil {
		return nil, err
	}
	repo, err := baseRepoFunc()
	if err != nil {
		// use default hostname if remote resolver fails
		repo = glrepo.NewWithHost("", "", glinstance.OverridableDefault())
	}
	return LabClientFunc(repo.RepoHost(), cfg, false)
}

// safeHttpClientFunc returns a function that ensures thread-safe initialization of the HTTP client
// to prevent race conditions when multiple goroutines attempt to create a client simultaneously
func (f *Factory) safeHttpClientFunc() func() (*gitlab.Client, error) {
	return func() (*gitlab.Client, error) {
		f.clientOnce.Do(func() {
			f.client, f.clientErr = httpClientFunc()
		})
		return f.client, f.clientErr
	}
}

func NewFactory() *Factory {
	f := &Factory{
		Config:     configFunc,
		Remotes:    remotesFunc,
		HttpClient: httpClientFunc,
		BaseRepo:   baseRepoFunc,
		Branch: func() (string, error) {
			currentBranch, err := git.CurrentBranch()
			if err != nil {
				return "", fmt.Errorf("could not determine current branch: %w", err)
			}
			return currentBranch, nil
		},
		IO: iostreams.Init(),
	}

	f.HttpClient = f.safeHttpClientFunc()
	return f
}
