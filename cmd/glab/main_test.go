//go:build !integration

package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"

	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/internal/testing/cmdtest"
)

// Test started when the test binary is started
// and calls the main function
func TestGlab(t *testing.T) { // nolint:unparam
	main()
}

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m,
		goleak.IgnoreTopFunction("internal/poll.runtime_pollWait"), // HTTP keep-alive connections
	)
}

func TestMain_isUpdateCheckEnabled(t *testing.T) {
	tests := []struct {
		name           string
		envValue       string
		config         config.Config
		expectedResult bool
		description    string
	}{
		{
			name:           "env var enabled true",
			envValue:       "true",
			config:         config.NewFromString(""),
			expectedResult: true,
			description:    "GLAB_CHECK_UPDATE=true should return true",
		},
		{
			name:           "env var enabled 1",
			envValue:       "1",
			config:         config.NewFromString(""),
			expectedResult: true,
			description:    "GLAB_CHECK_UPDATE=1 should return true",
		},
		{
			name:           "env var disabled false",
			envValue:       "false",
			config:         config.NewFromString(""),
			expectedResult: false,
			description:    "GLAB_CHECK_UPDATE=false should return false",
		},
		{
			name:           "env var disabled 0",
			envValue:       "0",
			config:         config.NewFromString(""),
			expectedResult: false,
			description:    "GLAB_CHECK_UPDATE=0 should return false",
		},
		{
			name:           "config enabled true",
			envValue:       "",
			config:         config.NewFromString("check_update: true"),
			expectedResult: true,
			description:    "check_update=true in config should return true",
		},
		{
			name:           "config enabled 1",
			envValue:       "",
			config:         config.NewFromString("check_update: 1"),
			expectedResult: true,
			description:    "check_update=1 in config should return true",
		},
		{
			name:           "config disabled false",
			envValue:       "",
			config:         config.NewFromString("check_update: false"),
			expectedResult: false,
			description:    "check_update=false in config should return false",
		},
		{
			name:           "config disabled 0",
			envValue:       "",
			config:         config.NewFromString("check_update: 0"),
			expectedResult: false,
			description:    "check_update=0 in config should return false",
		},
		{
			name:           "env var overrides config",
			envValue:       "false",
			config:         config.NewFromString("check_update: true"),
			expectedResult: false,
			description:    "Environment variable should take precedence over config",
		},
		{
			name:           "default when no config",
			envValue:       "",
			config:         config.NewFromString(""),
			expectedResult: true,
			description:    "Should default to true when no config is set",
		},
		{
			name:           "invalid config value returns false",
			envValue:       "",
			config:         config.NewFromString("check_update: invalid"),
			expectedResult: false,
			description:    "Invalid config value should return false (strconv.ParseBool zero value)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variable if specified
			if tt.envValue != "" {
				t.Setenv("GLAB_CHECK_UPDATE", tt.envValue)
			}

			ios, _, _, _ := cmdtest.TestIOStreams()
			factory := cmdtest.NewTestFactory(ios, cmdtest.WithConfig(tt.config))

			result := isUpdateCheckEnabled(factory)

			assert.Equal(t, tt.expectedResult, result, tt.description)
		})
	}
}
