package oauth2

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cli/internal/glinstance"
)

func TestWillExpire(t *testing.T) {
	tt := []struct {
		description string
		expiryDate  time.Time
		expected    bool
	}{
		{
			description: "Will expire in the future",
			expiryDate:  time.Now().Add(minimumTokenLifetime * 2),
			expected:    false,
		},
		{
			description: "Will expire soon",
			expiryDate:  time.Now().Add(minimumTokenLifetime - time.Minute),
			expected:    true,
		},
		{
			description: "Has expire in the past",
			expiryDate:  time.Now().Add(-10 * time.Minute),
			expected:    true,
		},
		{
			description: "Expires right now",
			expiryDate:  time.Now(),
			expected:    true,
		},
	}
	for _, test := range tt {
		t.Run(test.description, func(t *testing.T) {
			tokenNow := &AuthToken{
				ExpiryDate: test.expiryDate,
			}
			require.Equal(t, test.expected, tokenNow.WillExpire())
		})
	}
}

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
		"client_id":            "321",
	}

	stderr := &bytes.Buffer{}

	tokenCh := handleAuthRedirect(stderr, &http.Client{}, "localhost", "123", hostname, "http", "abc", "validstate")
	defer close(tokenCh)
	time.Sleep(1 * time.Second)

	tt := []struct {
		description         string
		state               string
		expectedToken       bool
		expectedAccessToken string
	}{
		{
			"when valid request then token is returned",
			"validstate",
			true,
			"at",
		},
		{
			"when invalid state is passed does not return token",
			"invalidstate",
			false,
			"",
		},
	}

	for _, test := range tt {
		t.Run(test.description, func(t *testing.T) {
			go func() {
				url := fmt.Sprintf("http://localhost:7171/auth/redirect?code=123&state=%s", test.state)
				_, err := http.Get(url)
				require.Nil(t, err)
			}()

			token := <-tokenCh
			if !test.expectedToken {
				require.Nil(t, token)
				return
			}

			assert.Equal(t, "at", token.AccessToken)
		})
	}
}

func TestClientID(t *testing.T) {
	testCasesTable := []struct {
		name             string
		hostname         string
		configClientID   string
		expectedClientID string
	}{
		{
			name:             "managed",
			hostname:         glinstance.DefaultHostname,
			configClientID:   "",
			expectedClientID: glinstance.DefaultClientID,
		},
		{
			name:             "self-managed-complete",
			hostname:         "salsa.debian.org",
			configClientID:   "321",
			expectedClientID: "321",
		},
	}

	for _, testCase := range testCasesTable {
		t.Run(testCase.name, func(t *testing.T) {
			cfg := stubConfig{
				hosts: map[string]map[string]string{
					testCase.hostname: {
						"client_id": testCase.configClientID,
					},
				},
			}
			clientID, err := oAuthClientID(cfg, testCase.hostname)
			assert.NoError(t, err)
			assert.Equal(t, testCase.expectedClientID, clientID)
		})
	}

	t.Run("invalid self-managed config", func(t *testing.T) {
		cfg := stubConfig{
			hosts: map[string]map[string]string{
				"salsa.debian.org": {},
			},
		}
		clientID, err := oAuthClientID(cfg, "salsa.debian.org")
		assert.Error(t, err)
		assert.Empty(t, clientID)
	})
}
