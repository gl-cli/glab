package oauth2

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cli/internal/config"
	"golang.org/x/oauth2"
)

type stubConfig struct {
	hosts map[string]map[string]string
}

func (s stubConfig) Get(host string, key string) (string, error) {
	return s.hosts[host][key], nil
}

func (s stubConfig) GetWithSource(string, string, bool) (string, string, error) { return "", "", nil }
func (s stubConfig) Set(host string, key string, value string) error {
	if _, ok := s.hosts[host]; !ok {
		s.hosts[host] = make(map[string]string)
	}
	s.hosts[host][key] = value

	return nil
}

func (s stubConfig) UnsetHost(string)                      {}
func (s stubConfig) Hosts() ([]string, error)              { return nil, nil }
func (s stubConfig) Aliases() (*config.AliasConfig, error) { return nil, nil }
func (s stubConfig) Local() (*config.LocalConfig, error)   { return nil, nil }
func (s stubConfig) Write() error                          { return nil }
func (s stubConfig) WriteAll() error                       { return nil }

func TestConfig_unmarshal(t *testing.T) {
	cfg := stubConfig{
		hosts: map[string]map[string]string{
			"gitlab.com": {
				"is_oauth2":            "true",
				"oauth2_refresh_token": "refresh_token",
				"token":                "access_token",
				"oauth2_expiry_date":   "13 Mar 23 15:47 GMT",
			},
		},
	}

	token, err := unmarshal("gitlab.com", cfg)
	require.Nil(t, err)

	assert.Equal(t, "refresh_token", token.RefreshToken)
	assert.Equal(t, "access_token", token.AccessToken)

	expectedDate, err := time.Parse(time.RFC822, "13 Mar 23 15:47 GMT")
	require.Nil(t, err)

	assert.Equal(t, expectedDate, token.Expiry)
}

func TestConfig_marshal(t *testing.T) {
	cfg := stubConfig{
		hosts: map[string]map[string]string{},
	}

	token := &oauth2.Token{
		RefreshToken: "refresh_token",
		AccessToken:  "access_token",
		ExpiresIn:    60,
		Expiry:       time.Now().Add(60 * time.Second),
	}

	expectedDate := time.Now().Add(60 * time.Second)

	err := marshal("gitlab.com", cfg, token)
	require.Nil(t, err)

	require.Equal(t, cfg.hosts, map[string]map[string]string{
		"gitlab.com": {
			"is_oauth2":            "true",
			"oauth2_refresh_token": "refresh_token",
			"token":                "access_token",
			"oauth2_expiry_date":   expectedDate.Format(time.RFC822),
		},
	})
}
