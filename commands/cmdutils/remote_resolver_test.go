package cmdutils

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/pkg/git"

	"github.com/MakeNowJust/heredoc/v2"
)

func Test_remoteResolver(t *testing.T) {
	rr := &remoteResolver{
		readRemotes: func() (git.RemoteSet, error) {
			return git.RemoteSet{
				git.NewRemote("fork", "https://example.org/owner/fork.git"),
				git.NewRemote("origin", "https://gitlab.com/owner/repo.git"),
				git.NewRemote("upstream", "https://example.org/owner/repo.git"),
			}, nil
		},
		getConfig: func() (config.Config, error) {
			return config.NewFromString(heredoc.Doc(`
				hosts:
				  example.org:
				    oauth_token: OTOKEN
			`)), nil
		},
		urlTranslator: func(u *url.URL) *url.URL {
			return u
		},
	}

	resolver := rr.Resolver("")
	remotes, err := resolver()
	require.NoError(t, err)
	require.Equal(t, 2, len(remotes))

	assert.Equal(t, "upstream", remotes[0].Name)
	assert.Equal(t, "fork", remotes[1].Name)
}

func Test_remoteResolverOverride(t *testing.T) {
	rr := &remoteResolver{
		readRemotes: func() (git.RemoteSet, error) {
			return git.RemoteSet{
				git.NewRemote("fork", "https://example.org/ghe-owner/ghe-fork.git"),
				git.NewRemote("origin", "https://gitlab.com/owner/repo.git"),
				git.NewRemote("upstream", "https://example.org/ghe-owner/ghe-repo.git"),
			}, nil
		},
		getConfig: func() (config.Config, error) {
			return config.NewFromString(heredoc.Doc(`
				hosts:
				  example.org:
				    oauth_token: GHETOKEN
			`)), nil
		},
		urlTranslator: func(u *url.URL) *url.URL {
			return u
		},
	}

	resolver := rr.Resolver("gitlab.com")
	remotes, err := resolver()
	require.NoError(t, err)
	require.Equal(t, 1, len(remotes))

	assert.Equal(t, "origin", remotes[0].Name)
}

func Test_remoteResolverErrors(t *testing.T) {
	testRemotes := git.RemoteSet{
		git.NewRemote("origin", "https://example3.org/owner/fork.git"),
		git.NewRemote("fork", "https://example.org/owner/fork.git"),
		git.NewRemote("upstream", "https://example.org/owner/repo.git"),
		git.NewRemote("foo", "https://example2.org/owner/repo.git"),
	}

	tests := []struct {
		name          string
		remotes       git.RemoteSet
		hostOverride  string
		expectedError string
	}{
		{
			name:          "No remotes",
			remotes:       git.RemoteSet{},
			expectedError: "no git remotes found",
		},
		{
			name:         "No match with host override",
			remotes:      testRemotes,
			hostOverride: "nomatch.org",
			expectedError: "none of the git remotes configured for this repository correspond to the GITLAB_HOST environment variable. " +
				"Try adding a matching remote or unsetting the variable.\n\n" +
				"GITLAB_HOST is currently set to nomatch.org\n\n" +
				"Configured remotes: example.org, example3.org, example2.org",
		},
		{
			name:    "No match",
			remotes: testRemotes,
			expectedError: "none of the git remotes configured for this repository point to a known GitLab host. " +
				"Please use `glab auth login` to authenticate and configure a new host for glab.\n\n" +
				"Configured remotes: example.org, example3.org, example2.org",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rr := &remoteResolver{
				readRemotes: func() (git.RemoteSet, error) {
					return test.remotes, nil
				},
				getConfig: func() (config.Config, error) {
					return config.NewFromString(heredoc.Doc(`
				hosts:
				  my-gitlab.org:
				    oauth_token: OTOKEN
			`)), nil
				},
			}

			resolver := rr.Resolver(test.hostOverride)
			_, err := resolver()
			require.Error(t, err)
			assert.Equal(t, test.expectedError, err.Error())
		})
	}
}
