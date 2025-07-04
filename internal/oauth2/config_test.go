package oauth2

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

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
