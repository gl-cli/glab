package docker

import (
	"testing"
	"time"

	"github.com/docker/docker-credential-helpers/credentials"
	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/cli/internal/config"
)

func TestHelper(t *testing.T) {
	// This avoids the oauth2 refresh from sending an http request.
	futureDate := time.Now().Add(24 * time.Hour).Format(time.RFC822)

	t.Run("Get", func(t *testing.T) {
		// This ensures that we don't pull the wrong user or token
		// if the config file search is incorrectly done with
		// env variable search space included.
		t.Setenv("USER", "wrong_user")
		t.Setenv("GITLAB_TOKEN", "wrong_token")

		t.Run("without error", func(t *testing.T) {
			tests := map[string]struct {
				cfg            config.Config
				registryURL    string
				expectUser     string
				expectPassword string
			}{
				"single host": {
					cfg: config.NewFromString(`
---
hosts:
  gitlab.example.com:
    is_oauth2: "true"
    client_id: abc
    user: user1
    token: token1
    oauth2_expiry_date: ` + futureDate + `
    container_registry_domains: registry.gitlab.example.com
`),
					registryURL:    "registry.gitlab.example.com",
					expectUser:     "user1",
					expectPassword: "token1",
				},
				"multi-host": {
					cfg: config.NewFromString(`
---
hosts:
  gitlab.example.com:
    is_oauth2: "true"
    client_id: abc
    user: user1
    token: token1
    oauth2_expiry_date: ` + futureDate + `
    container_registry_domains: registry.gitlab.example.com
  gdk.example.com:
    is_oauth2: "true"
    client_id: abc
    user: user2
    token: token2
    oauth2_expiry_date: ` + futureDate + `
    container_registry_domains: registry.gdk.example.com
`),
					registryURL:    "registry.gdk.example.com",
					expectUser:     "user2",
					expectPassword: "token2",
				},
			}

			for name, tt := range tests {
				t.Run(name, func(t *testing.T) {
					t.Parallel()
					helper := Helper{cfg: tt.cfg}
					gotUser, gotPassword, err := helper.Get(tt.registryURL)
					assert.NoError(t, err)
					assert.Equal(t, tt.expectUser, gotUser, "username does not match")
					assert.Equal(t, tt.expectPassword, gotPassword, "password does not match")
				})
			}
		})

		t.Run("with error", func(t *testing.T) {
			tests := map[string]struct {
				cfg         config.Config
				registryURL string
				expectErr   string
			}{
				"no associated hostname": {
					cfg: config.NewFromString(`
---
hosts:
  gitlab.example.com:
    is_oauth2: "true"
    client_id: abc
    user: user1
    token: token1
    oauth2_expiry_date: ` + futureDate + `
`),
					registryURL: "gitlab.example.com",
					expectErr:   "no hostname associated with",
				},
				"empty username": {
					cfg: config.NewFromString(`
---
hosts:
  gitlab.example.com:
    is_oauth2: "true"
    client_id: abc
    user: ""
    token: token1
    oauth2_expiry_date: ` + futureDate + `
    container_registry_domains: registry.gitlab.example.com
`),
					registryURL: "registry.gitlab.example.com",
					expectErr:   "glab user for this registryURL (hostname) is empty",
				},
				"empty token": {
					cfg: config.NewFromString(`
---
hosts:
  gitlab.example.com:
    is_oauth2: "true"
    client_id: abc
    user: user1
    token: ""
    oauth2_expiry_date: ` + futureDate + `
    container_registry_domains: registry.gitlab.example.com
`),
					registryURL: "registry.gitlab.example.com",
					expectErr:   "glab token for this registryURL (hostname) is empty",
				},
				"no username": {
					cfg: config.NewFromString(`
---
hosts:
  gitlab.example.com:
    is_oauth2: "true"
    client_id: abc
    token: token1
    oauth2_expiry_date: ` + futureDate + `
    container_registry_domains: registry.gitlab.example.com
`),
					registryURL: "registry.gitlab.example.com",
					expectErr:   "glab user for this registryURL (hostname) is empty",
				},
				"no token": {
					cfg: config.NewFromString(`
---
hosts:
  gitlab.example.com:
    is_oauth2: "true"
    client_id: abc
    user: user1
    oauth2_expiry_date: ` + futureDate + `
    container_registry_domains: registry.gitlab.example.com
`),
					registryURL: "registry.gitlab.example.com",
					expectErr:   "glab token for this registryURL (hostname) is empty",
				},
			}

			for name, tt := range tests {
				t.Run(name, func(t *testing.T) {
					t.Parallel()
					helper := Helper{cfg: tt.cfg}
					gotUser, gotPassword, err := helper.Get(tt.registryURL)
					assert.ErrorContains(t, err, tt.expectErr)
					assert.Empty(t, gotUser, "username is not empty")
					assert.Empty(t, gotPassword, "password is not empty")
				})
			}
		})
	})

	t.Run("Add", func(t *testing.T) {
		t.Parallel()
		var helper Helper
		err := helper.Add(&credentials.Credentials{})
		assert.ErrorContains(t, err, "glab auth docker-helper does not")
	})

	t.Run("Delete", func(t *testing.T) {
		t.Parallel()
		var helper Helper
		err := helper.Delete("registry.gitlab.example.com")
		assert.ErrorContains(t, err, "glab auth docker-helper does not")
	})

	t.Run("List", func(t *testing.T) {
		t.Parallel()
		var helper Helper
		got, err := helper.List()
		assert.ErrorContains(t, err, "glab auth docker-helper does not")
		assert.Empty(t, got)
	})
}
