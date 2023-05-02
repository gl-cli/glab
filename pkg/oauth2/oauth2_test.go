package oauth2

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cli/pkg/iostreams"
)

func TestHandleAuthRedirect(t *testing.T) {
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"access_token": "at",
			"refresh_token": "rt",
			"expiresIn": 60
		}`))
	}))

	cfg := stubConfig{
		hosts: map[string]map[string]string{},
	}

	hostname := strings.Split(svr.URL, "://")[1]
	cfg.hosts[hostname] = map[string]string{
		"is_oauth2":            "true",
		"oauth2_refresh_token": "refresh_token",
		"token":                "access_token",
		"oauth2_code_verifier": "123",
		"oauth2_expiry_date":   "13 Mar 23 15:47 GMT",
	}

	ios, _, _, _ := iostreams.Test()

	tokenCh := handleAuthRedirect(ios, "123", hostname, "http")
	defer close(tokenCh)
	time.Sleep(1 * time.Second)

	go func() {
		_, err := http.Get("http://localhost:7171/auth/redirect?code=123")
		require.Nil(t, err)
	}()

	token := <-tokenCh
	assert.Equal(t, "at", token.AccessToken)
}

func TestRefreshToken(t *testing.T) {
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"access_token": "at",
			"refresh_token": "rt",
			"expiresIn": 60
		}`))
	}))

	cfg := stubConfig{
		hosts: map[string]map[string]string{},
	}

	hostname := strings.Split(svr.URL, "://")[1]
	cfg.hosts[hostname] = map[string]string{
		"is_oauth2":            "true",
		"oauth2_refresh_token": "refresh_token",
		"token":                "access_token",
		"oauth2_code_verifier": "123",
		"oauth2_expiry_date":   "13 Mar 23 15:47 GMT",
	}

	err := RefreshToken(hostname, cfg, "http")
	require.Nil(t, err)

	accessToken, err := cfg.Get(hostname, "token")
	require.Nil(t, err)
	assert.Equal(t, "at", accessToken)

	refreshToken, err := cfg.Get(hostname, "oauth2_refresh_token")
	require.Nil(t, err)
	assert.Equal(t, "rt", refreshToken)

	expiryDateString, err := cfg.Get(hostname, "oauth2_expiry_date")
	require.Nil(t, err)
	_, err = time.Parse(time.RFC822, expiryDateString)
	require.Nil(t, err)
}
