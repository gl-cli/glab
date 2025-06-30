package oauth2

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCalcExpiresDate(t *testing.T) {
	token := AuthToken{ExpiresIn: 60}

	token.CalcExpiresDate()

	assert.True(t, time.Now().Add(1*time.Minute).After(token.ExpiryDate))
	assert.True(t, time.Now().Before(token.ExpiryDate))
}

func TestTokenFromConfig(t *testing.T) {
	cfg := stubConfig{
		hosts: map[string]map[string]string{
			"gitlab.com": {
				"is_oauth2":            "true",
				"oauth2_refresh_token": "refresh_token",
				"token":                "access_token",
				"oauth2_code_verifier": "123",
				"oauth2_expiry_date":   "13 Mar 23 15:47 GMT",
			},
		},
	}

	token, err := tokenFromConfig("gitlab.com", cfg)
	require.Nil(t, err)

	assert.Equal(t, "refresh_token", token.RefreshToken)
	assert.Equal(t, "access_token", token.AccessToken)
	assert.Equal(t, "123", token.CodeVerifier)

	expectedDate, err := time.Parse(time.RFC822, "13 Mar 23 15:47 GMT")
	require.Nil(t, err)

	assert.Equal(t, expectedDate, token.ExpiryDate)
}

func TestTokenSetConfig(t *testing.T) {
	cfg := stubConfig{
		hosts: map[string]map[string]string{},
	}

	token := AuthToken{
		RefreshToken: "refresh_token",
		AccessToken:  "access_token",
		CodeVerifier: "123",
		ExpiresIn:    60,
	}

	expectedDate := time.Now().Add(60 * time.Second)

	err := token.SetConfig("gitlab.com", cfg)
	require.Nil(t, err)

	require.Equal(t, cfg.hosts, map[string]map[string]string{
		"gitlab.com": {
			"is_oauth2":            "true",
			"oauth2_refresh_token": "refresh_token",
			"token":                "access_token",
			"oauth2_code_verifier": "123",
			"oauth2_expiry_date":   expectedDate.Format(time.RFC822),
		},
	})
}
