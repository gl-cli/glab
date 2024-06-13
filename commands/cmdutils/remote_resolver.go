package cmdutils

import (
	"errors"
	"net/url"
	"sort"
	"strings"

	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/pkg/git"
	"gitlab.com/gitlab-org/cli/pkg/glinstance"
)

type remoteResolver struct {
	readRemotes   func() (git.RemoteSet, error)
	getConfig     func() (config.Config, error)
	urlTranslator func(*url.URL) *url.URL
}

func (rr *remoteResolver) Resolver(hostOverride string) func() (glrepo.Remotes, error) {
	var cachedRemotes glrepo.Remotes
	var remotesError error

	return func() (glrepo.Remotes, error) {
		if cachedRemotes != nil || remotesError != nil {
			return cachedRemotes, remotesError
		}

		gitRemotes, err := rr.readRemotes()
		if err != nil {
			remotesError = err
			return nil, err
		}
		if len(gitRemotes) == 0 {
			remotesError = errors.New("no git remotes found")
			return nil, remotesError
		}

		sshTranslate := rr.urlTranslator
		if sshTranslate == nil {
			sshTranslate = git.ParseSSHConfig().Translator()
		}
		resolvedRemotes := glrepo.TranslateRemotes(gitRemotes, sshTranslate)

		cfg, err := rr.getConfig()
		if err != nil {
			return nil, err
		}

		knownHosts := map[string]bool{}
		knownHosts[glinstance.Default()] = true
		if authenticatedHosts, err := cfg.Hosts(); err == nil {
			for _, h := range authenticatedHosts {
				knownHosts[h] = true
			}
		}

		// filter remotes to only those sharing a single, known hostname
		var hostname string
		cachedRemotes = glrepo.Remotes{}
		sort.Sort(resolvedRemotes)

		if hostOverride != "" {
			for _, r := range resolvedRemotes {
				if strings.EqualFold(r.RepoHost(), hostOverride) {
					cachedRemotes = append(cachedRemotes, r)
				}
			}

			if len(cachedRemotes) == 0 {
				remotesError = errors.New("none of the git remotes configured for this repository correspond to the GITLAB_HOST environment variable. Try adding a matching remote or unsetting the variable.\n\n" +
					"GITLAB_HOST is currently set to " + hostOverride + "\n\nConfigured remotes: " + resolvedRemotes.UniqueHosts())
				return nil, remotesError
			}

			return cachedRemotes, nil
		}

		for _, r := range resolvedRemotes {
			if hostname == "" {
				if !knownHosts[r.RepoHost()] {
					continue
				}
				hostname = r.RepoHost()
			} else if r.RepoHost() != hostname {
				continue
			}
			cachedRemotes = append(cachedRemotes, r)
		}

		if len(cachedRemotes) == 0 {
			remotesError = errors.New("none of the git remotes configured for this repository point to a known GitLab host. Please use `glab auth login` to authenticate and configure a new host for glab.\n\n" +
				"Configured remotes: " + resolvedRemotes.UniqueHosts())
			return nil, remotesError
		}
		return cachedRemotes, nil
	}
}
