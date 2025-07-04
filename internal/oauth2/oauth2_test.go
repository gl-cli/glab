package oauth2

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/cli/internal/glinstance"
)

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
