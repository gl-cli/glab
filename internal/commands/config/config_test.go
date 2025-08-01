package config

import (
	"errors"
	"testing"

	"gitlab.com/gitlab-org/cli/internal/testing/cmdtest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cli/internal/config"
)

type configStub map[string]string

func (c configStub) Local() (*config.LocalConfig, error) {
	return nil, nil
}

func (c configStub) WriteAll() error {
	c["_written"] = "true"
	return nil
}

func genKey(host, key string) string {
	if host != "" {
		return host + ":" + key
	}
	return key
}

func (c configStub) Get(host, key string) (string, error) {
	val, _, err := c.GetWithSource(host, key, true)
	return val, err
}

func (c configStub) GetWithSource(host, key string, searchENVVars bool) (string, string, error) {
	if v, found := c[genKey(host, key)]; found {
		return v, "(memory)", nil
	}
	return "", "", errors.New("not found")
}

func (c configStub) Set(host, key, value string) error {
	c[genKey(host, key)] = value
	return nil
}

func (c configStub) Aliases() (*config.AliasConfig, error) {
	return nil, nil
}

func (c configStub) Hosts() ([]string, error) {
	return nil, nil
}

func (c configStub) Write() error {
	c["_written"] = "true"
	return nil
}

func TestConfigGet(t *testing.T) {
	tests := []struct {
		name   string
		config configStub
		args   []string
		stdout string
		stderr string
		isTTY  bool
	}{
		{
			name: "get key",
			config: configStub{
				"editor": "ed",
			},
			args:   []string{"editor"},
			stdout: "ed\n",
			stderr: "",
			isTTY:  true,
		},
		{
			name: "get key scoped by host",
			config: configStub{
				"editor":            "ed",
				"gitlab.com:editor": "vim",
			},
			args:   []string{"editor", "--host", "gitlab.com"},
			stdout: "vim\n",
			stderr: "",
			isTTY:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			io, _, stdout, stderr := cmdtest.TestIOStreams(cmdtest.WithTestIOStreamsAsTTY(tt.isTTY))

			f := cmdtest.NewTestFactory(io,
				cmdtest.WithConfig(tt.config),
			)

			cmd := NewCmdConfigGet(f)
			cmd.Flags().BoolP("help", "x", false, "")
			cmd.SetArgs(tt.args)
			cmd.SetOut(stdout)
			cmd.SetErr(stderr)

			_, err := cmd.ExecuteC()
			require.NoError(t, err)

			assert.Equal(t, tt.stdout, stdout.String())
			assert.Equal(t, tt.stderr, stderr.String())
			assert.Equal(t, "", tt.config["_written"])
		})
	}
}

func TestConfigSet(t *testing.T) {
	tests := []struct {
		name      string
		config    configStub
		args      []string
		expectKey string
		stdout    string
		stderr    string
		isTTY     bool
	}{
		{
			name:      "set key",
			config:    configStub{},
			args:      []string{"editor", "vim"},
			expectKey: "editor",
			stdout:    "",
			stderr:    "",
			isTTY:     true,
		},
		{
			name:      "set key scoped by host",
			config:    configStub{},
			args:      []string{"editor", "vim", "--host", "gitlab.com"},
			expectKey: "gitlab.com:editor",
			stdout:    "",
			stderr:    "",
			isTTY:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			io, _, stdout, stderr := cmdtest.TestIOStreams(cmdtest.WithTestIOStreamsAsTTY(tt.isTTY))

			f := cmdtest.NewTestFactory(io,
				cmdtest.WithConfig(tt.config),
			)

			cmd := NewCmdConfigSet(f)
			cmd.Flags().BoolP("help", "x", false, "")
			cmd.SetArgs(append(tt.args, "-g"))
			cmd.SetOut(stdout)
			cmd.SetErr(stderr)

			_, err := cmd.ExecuteC()
			require.NoError(t, err)

			assert.Equal(t, tt.stdout, stdout.String())
			assert.Equal(t, tt.stderr, stderr.String())
			assert.Equal(t, "vim", tt.config[tt.expectKey])
			assert.Equal(t, "true", tt.config["_written"])
		})
	}
}
