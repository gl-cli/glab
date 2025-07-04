package oauth2

import (
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
			clientID, err := oauthClientID(cfg, testCase.hostname)
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
		clientID, err := oauthClientID(cfg, "salsa.debian.org")
		assert.Error(t, err)
		assert.Empty(t, clientID)
	})
}
