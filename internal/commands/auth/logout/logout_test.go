package logout

import (
	"bytes"
	"fmt"
	"io"
	"testing"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/stretchr/testify/assert"

	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/internal/testing/cmdtest"
	"gitlab.com/gitlab-org/cli/test"
)

func runCommand(cfg config.Config, args string) (*test.CmdOut, error) {
	ios, _, stdout, stderr := cmdtest.TestIOStreams(cmdtest.WithTestIOStreamsAsTTY(true))
	factory := cmdtest.NewTestFactory(ios, cmdtest.WithConfig(cfg))

	cmd := NewCmdLogout(factory)
	// workaround for CI
	cmd.Flags().BoolP("help", "x", false, "")

	return cmdtest.ExecuteCommand(cmd, args, stdout, stderr)
}

func Test_NewCmdLogout(t *testing.T) {
	tests := []struct {
		name     string
		hostname string
		stdErr   string
		wantErr  bool
	}{
		{
			name:     "no arguments",
			hostname: "",
			stdErr:   "hostname is required to logout. Use --hostname flag to specify hostname",
			wantErr:  true,
		},
		{
			name:     "hostname set",
			hostname: "gitlab.example.com",
			stdErr:   "",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mainBuf := bytes.Buffer{}
			defer config.StubWriteConfig(&mainBuf, io.Discard)()

			token := "xxxxxxxx"

			cfg := config.NewFromString(heredoc.Docf(
				`
					hosts:
					  gitlab.something.com:
					    token: %[1]s
					  gitlab.example.com:
					    token: %[1]s
				`,
				token,
			))

			// removing the environment variable so CI does not interfere
			t.Setenv("GITLAB_TOKEN", "")

			output, err := runCommand(cfg, fmt.Sprintf("--hostname %s", tt.hostname))

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				logoutMessage := fmt.Sprintf("Successfully logged out of %s\n", tt.hostname)
				assert.Equal(t, logoutMessage, output.String())

				cfg := config.NewFromString(mainBuf.String())
				gitlabToken, _ := cfg.Get("gitlab.something.com", "token")
				assert.Equal(t, token, gitlabToken)

				exampleToken, _ := cfg.Get(tt.hostname, "token")
				assert.Equal(t, "", exampleToken)
			}
		})
	}
}
