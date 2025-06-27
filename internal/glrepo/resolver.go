package glrepo

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"gitlab.com/gitlab-org/cli/internal/api"
	"gitlab.com/gitlab-org/cli/internal/git"
	"gitlab.com/gitlab-org/cli/internal/prompt"

	"github.com/hashicorp/go-multierror"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// cap the number of git remotes looked up, since the user might have an
// unusually large number of git remotes
const maxRemotesForLookup = 5

func ResolveRemotesToRepos(remotes Remotes, client *gitlab.Client, defaultHostname string) (*ResolvedRemotes, error) {
	sort.Stable(remotes)

	result := &ResolvedRemotes{
		remotes:         remotes,
		apiClient:       client,
		defaultHostname: defaultHostname,
	}

	return result, nil
}

func resolveNetwork(result *ResolvedRemotes) error {
	// Loop over at most 5 (maxRemotesForLookup)
	var errs error
	anySuccess := false
	for i := 0; i < len(result.remotes) && i < maxRemotesForLookup; i++ {
		networkResult, err := api.GetProject(result.apiClient, result.remotes[i].FullName())
		if err == nil {
			result.network = append(result.network, *networkResult)
			anySuccess = true
		} else {
			errs = multierror.Append(errs, fmt.Errorf("%s: %w", result.remotes[i].FullName(), err))
		}
	}
	if anySuccess {
		return nil
	}
	return errs
}

type ResolvedRemotes struct {
	remotes         Remotes
	network         []gitlab.Project
	apiClient       *gitlab.Client
	defaultHostname string
}

func (r *ResolvedRemotes) BaseRepo(interactive bool) (Interface, error) {
	// if any of the remotes already has a resolution, respect that
	for _, remote := range r.remotes {
		if remote.Resolved == "base" {
			return remote, nil
		} else if strings.HasPrefix(remote.Resolved, "base:") {
			repo, err := FromFullName(strings.TrimPrefix(remote.Resolved, "base:"), r.defaultHostname)
			if err != nil {
				return nil, err
			}
			return NewWithHost(repo.RepoOwner(), repo.RepoName(), remote.RepoHost()), nil
		} else if remote.Resolved != "" && !strings.HasPrefix(remote.Resolved, "head") {
			// Backward compatibility kludge for remote-less resolutions created before
			// BaseRepo started creating resolutions prefixed with `base:`
			repo, err := FromFullName(remote.Resolved, r.defaultHostname)
			if err != nil {
				return nil, err
			}
			// Rewrite resolution, ignore the error as this will keep working
			// in the future we might add a warning that we couldn't rewrite
			// it for compatibility
			_ = git.SetRemoteResolution(remote.Name, "base:"+remote.Resolved)

			return NewWithHost(repo.RepoOwner(), repo.RepoName(), remote.RepoHost()), nil
		}
	}

	if !interactive {
		// we cannot prompt, so just resort to the 1st remote
		return r.remotes[0], nil
	}

	// from here on, consult the API
	if r.network == nil {
		err := resolveNetwork(r)
		if err != nil {
			return nil, err
		}
		if len(r.network) == 0 {
			return nil, errors.New("no GitLab Projects found from remotes")
		}
	}

	var repoNames []string
	repoMap := map[string]*gitlab.Project{}
	add := func(r *gitlab.Project) {
		fn, _ := FullNameFromURL(r.HTTPURLToRepo)
		if _, ok := repoMap[fn]; !ok {
			repoMap[fn] = r
			repoNames = append(repoNames, fn)
		}
	}

	for i := range r.network {
		if r.network[i].ForkedFromProject != nil {
			fProject, _ := api.GetProject(r.apiClient, r.network[i].ForkedFromProject.PathWithNamespace)
			add(fProject)
		}
		add(&r.network[i])
	}

	baseName := repoNames[0]
	if len(repoNames) > 1 {
		err := prompt.Select(
			&baseName,
			"base",
			"Which should be the base repository (used for e.g. querying issues) for this directory?",
			repoNames,
		)
		if err != nil {
			return nil, err
		}
	}

	// determine corresponding git remote
	selectedRepo := repoMap[baseName]
	selectedRepoInfo, _ := FromFullName(selectedRepo.HTTPURLToRepo, r.defaultHostname)
	resolution := "base"
	remote, _ := r.RemoteForRepo(selectedRepoInfo)
	if remote == nil {
		remote = r.remotes[0]
		resolution, _ = FullNameFromURL(selectedRepo.HTTPURLToRepo)
		resolution = "base:" + resolution
	}

	// cache the result to git config
	err := git.SetRemoteResolution(remote.Name, resolution)
	return selectedRepoInfo, err
}

func (r *ResolvedRemotes) HeadRepo(interactive bool) (Interface, error) {
	// if any of the remotes already has a resolution, respect that
	for _, remote := range r.remotes {
		if remote.Resolved == "head" {
			return remote, nil
		} else if strings.HasPrefix(remote.Resolved, "head:") {
			repo, err := FromFullName(strings.TrimPrefix(remote.Resolved, "head:"), r.defaultHostname)
			if err != nil {
				return nil, err
			}
			return NewWithHost(repo.RepoOwner(), repo.RepoName(), remote.RepoHost()), nil
		}
	}

	// from here on, consult the API
	if r.network == nil {
		err := resolveNetwork(r)
		if err != nil {
			return nil, err
		}
		if len(r.network) == 0 {
			return nil, errors.New("no GitLab Projects found from remotes")
		}
	}

	var repoNames []string
	repoMap := map[string]*gitlab.Project{}
	add := func(r *gitlab.Project) {
		fn, _ := FullNameFromURL(r.HTTPURLToRepo)
		if _, ok := repoMap[fn]; !ok {
			repoMap[fn] = r
			repoNames = append(repoNames, fn)
		}
	}

	for i := range r.network {
		if r.network[i].ForkedFromProject != nil {
			fProject, _ := api.GetProject(r.apiClient, r.network[i].ForkedFromProject.PathWithNamespace)
			add(fProject)
		}
		add(&r.network[i])
	}

	headName := repoNames[0]
	if len(repoNames) > 1 {
		if !interactive {
			// We cannot prompt so get the first repo that is a fork
			for _, repo := range repoNames {
				if repoMap[repo].ForkedFromProject != nil {
					selectedRepoInfo, _ := FromFullName((repoMap[repo].HTTPURLToRepo), r.defaultHostname)
					remote, _ := r.RemoteForRepo(selectedRepoInfo)
					return remote, nil
				}
			}
			// There are no forked repos so return the first repo
			return r.remotes[0], nil
		}

		err := prompt.Select(
			&headName,
			"head",
			"Which should be the head repository (where branches are pushed) for this directory?",
			repoNames,
		)
		if err != nil {
			return nil, err
		}
	}

	// determine corresponding git remote
	selectedRepo := repoMap[headName]
	selectedRepoInfo, _ := FromFullName(selectedRepo.HTTPURLToRepo, r.defaultHostname)
	resolution := "head"
	remote, _ := r.RemoteForRepo(selectedRepoInfo)
	if remote == nil {
		remote = r.remotes[0]
		resolution, _ = FullNameFromURL(selectedRepo.HTTPURLToRepo)
		resolution = "head:" + resolution
	}

	// cache the result to git config
	err := git.SetRemoteResolution(remote.Name, resolution)
	return selectedRepoInfo, err
}

// RemoteForRepo finds the git remote that points to a repository
func (r *ResolvedRemotes) RemoteForRepo(repo Interface) (*Remote, error) {
	for _, remote := range r.remotes {
		if IsSame(remote, repo) {
			return remote, nil
		}
	}
	return nil, errors.New("not found")
}
