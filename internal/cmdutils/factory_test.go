//go:build !integration

package cmdutils

import (
	"net/url"
	"testing"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/cli/internal/api"
	"gitlab.com/gitlab-org/cli/internal/config"
)

func TestFactory_ResolveHostNameFromConfig(t *testing.T) {
	// GIVEN
	cfg := config.NewFromString(heredoc.Doc(`
		host: gitlab.example.com
	`))

	// WHEN
	f := NewFactory(nil, false, cfg, api.BuildInfo{})

	// THEN
	assert.Equal(t, "gitlab.example.com", f.defaultHostname)
}

func TestFactory_ResolveHostNameFromEnv(t *testing.T) {
	// GIVEN
	t.Setenv("GITLAB_HOST", "gitlab.example.com")
	cfg := config.NewFromString(heredoc.Doc(`
		host: another.gitlab.example.com
	`))

	// WHEN
	f := NewFactory(nil, false, cfg, api.BuildInfo{})

	// THEN
	assert.Equal(t, "gitlab.example.com", f.defaultHostname)
}

func TestFactory_ResolveToGitLabComByDefault(t *testing.T) {
	// GIVEN
	cfg := config.NewBlankConfig()

	// WHEN
	f := NewFactory(nil, false, cfg, api.BuildInfo{})

	// THEN
	assert.Equal(t, "gitlab.com", f.defaultHostname)
}

func TestFactory_GitLabClientUsesCorrectHost(t *testing.T) {
	// GIVEN
	tests := []struct {
		name                         string
		cfg                          config.Config
		env                          map[string]string
		expectedGitLabClientHostname *url.URL
	}{
		{
			name:                         "default",
			cfg:                          config.NewBlankConfig(),
			expectedGitLabClientHostname: mustURL(t, "https://gitlab.com/api/v4/"),
		},
		{
			name: "host from config",
			cfg: config.NewFromString(heredoc.Doc(`
				host: gitlab.example.com
			`)),
			expectedGitLabClientHostname: mustURL(t, "https://gitlab.example.com/api/v4/"),
		},
		{
			name: "host from env",
			env:  map[string]string{"GITLAB_HOST": "gitlab.example.com"},
			cfg: config.NewFromString(heredoc.Doc(`
				host: another.gitlab.example.com
			`)),
			expectedGitLabClientHostname: mustURL(t, "https://gitlab.example.com/api/v4/"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// GIVEN
			if tt.env != nil {
				for k, v := range tt.env {
					t.Setenv(k, v)
				}
			}

			// WHEN
			f := NewFactory(nil, false, tt.cfg, api.BuildInfo{})
			c, err := f.GitLabClient()
			require.NoError(t, err)

			// THEN
			assert.Equal(t, tt.expectedGitLabClientHostname, c.BaseURL())
		})
	}
}

func mustURL(t *testing.T, s string) *url.URL {
	t.Helper()

	u, err := url.Parse(s)
	require.NoError(t, err)

	return u
}
