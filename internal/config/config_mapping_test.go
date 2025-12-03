package config

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEnvKeyEquivalence(t *testing.T) {
	tests := []struct {
		autologinEnabled bool
		inCi             bool
		givenKey         string
		expectedKeys     []string
	}{
		{
			autologinEnabled: false,
			inCi:             false,
			givenKey:         "api_host",
			expectedKeys:     []string{"GITLAB_API_HOST"},
		},
		{
			autologinEnabled: true,
			inCi:             false,
			givenKey:         "api_host",
			expectedKeys:     []string{"GITLAB_API_HOST"},
		},
		{
			autologinEnabled: false,
			inCi:             true,
			givenKey:         "api_host",
			expectedKeys:     []string{"GITLAB_API_HOST"},
		},
		{
			autologinEnabled: true,
			inCi:             true,
			givenKey:         "api_host",
			expectedKeys:     []string{"GITLAB_API_HOST", "CI_SERVER_FQDN"},
		},
		{
			autologinEnabled: false,
			inCi:             false,
			givenKey:         "api_protocol",
			expectedKeys:     []string{"API_PROTOCOL"},
		},
		{
			autologinEnabled: true,
			inCi:             false,
			givenKey:         "api_protocol",
			expectedKeys:     []string{"API_PROTOCOL"},
		},
		{
			autologinEnabled: false,
			inCi:             true,
			givenKey:         "api_protocol",
			expectedKeys:     []string{"API_PROTOCOL"},
		},
		{
			autologinEnabled: true,
			inCi:             true,
			givenKey:         "api_protocol",
			expectedKeys:     []string{"API_PROTOCOL", "CI_SERVER_PROTOCOL"},
		},
		{
			autologinEnabled: false,
			inCi:             false,
			givenKey:         "host",
			expectedKeys:     []string{"GITLAB_HOST", "GITLAB_URI", "GL_HOST"},
		},
		{
			autologinEnabled: true,
			inCi:             false,
			givenKey:         "host",
			expectedKeys:     []string{"GITLAB_HOST", "GITLAB_URI", "GL_HOST"},
		},
		{
			autologinEnabled: false,
			inCi:             true,
			givenKey:         "host",
			expectedKeys:     []string{"GITLAB_HOST", "GITLAB_URI", "GL_HOST"},
		},
		{
			autologinEnabled: true,
			inCi:             true,
			givenKey:         "host",
			expectedKeys:     []string{"GITLAB_HOST", "GITLAB_URI", "GL_HOST", "CI_SERVER_FQDN"},
		},
		{
			autologinEnabled: false,
			inCi:             false,
			givenKey:         "job_token",
			expectedKeys:     []string{"JOB_TOKEN"},
		},
		{
			autologinEnabled: true,
			inCi:             false,
			givenKey:         "job_token",
			expectedKeys:     []string{"JOB_TOKEN"},
		},
		{
			autologinEnabled: false,
			inCi:             true,
			givenKey:         "job_token",
			expectedKeys:     []string{"JOB_TOKEN"},
		},
		{
			autologinEnabled: true,
			inCi:             true,
			givenKey:         "job_token",
			expectedKeys:     []string{"JOB_TOKEN", "CI_JOB_TOKEN"},
		},
		{
			autologinEnabled: false,
			inCi:             false,
			givenKey:         "ca_cert",
			expectedKeys:     []string{"CA_CERT"},
		},
		{
			autologinEnabled: true,
			inCi:             false,
			givenKey:         "ca_cert",
			expectedKeys:     []string{"CA_CERT"},
		},
		{
			autologinEnabled: false,
			inCi:             true,
			givenKey:         "ca_cert",
			expectedKeys:     []string{"CA_CERT"},
		},
		{
			autologinEnabled: true,
			inCi:             true,
			givenKey:         "ca_cert",
			expectedKeys:     []string{"CA_CERT", "CI_SERVER_TLS_CA_FILE"},
		},
		{
			autologinEnabled: false,
			inCi:             false,
			givenKey:         "client_cert",
			expectedKeys:     []string{"CLIENT_CERT"},
		},
		{
			autologinEnabled: true,
			inCi:             false,
			givenKey:         "client_cert",
			expectedKeys:     []string{"CLIENT_CERT"},
		},
		{
			autologinEnabled: false,
			inCi:             true,
			givenKey:         "client_cert",
			expectedKeys:     []string{"CLIENT_CERT"},
		},
		{
			autologinEnabled: true,
			inCi:             true,
			givenKey:         "client_cert",
			expectedKeys:     []string{"CLIENT_CERT", "CI_SERVER_TLS_CERT_FILE"},
		},
		{
			autologinEnabled: false,
			inCi:             false,
			givenKey:         "client_key",
			expectedKeys:     []string{"CLIENT_KEY"},
		},
		{
			autologinEnabled: true,
			inCi:             false,
			givenKey:         "client_key",
			expectedKeys:     []string{"CLIENT_KEY"},
		},
		{
			autologinEnabled: false,
			inCi:             true,
			givenKey:         "client_key",
			expectedKeys:     []string{"CLIENT_KEY"},
		},
		{
			autologinEnabled: true,
			inCi:             true,
			givenKey:         "client_key",
			expectedKeys:     []string{"CLIENT_KEY", "CI_SERVER_TLS_KEY_FILE"},
		},
	}

	// clear potentially set keys that we use during tests
	t.Setenv("GLAB_ENABLE_CI_AUTOLOGIN", "")
	t.Setenv("GITLAB_CI", "")

	for _, tt := range tests {
		t.Run(fmt.Sprintf("autologin=%t ci=%t keys=%s -> %v", tt.autologinEnabled, tt.inCi, tt.givenKey, tt.expectedKeys), func(t *testing.T) {
			if tt.autologinEnabled {
				t.Setenv("GLAB_ENABLE_CI_AUTOLOGIN", "true")
			}
			if tt.inCi {
				t.Setenv("GITLAB_CI", "true")
			}

			actualKeys := EnvKeyEquivalence(tt.givenKey)

			assert.ElementsMatch(t, tt.expectedKeys, actualKeys)
		})
	}
}
