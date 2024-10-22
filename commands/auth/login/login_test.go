package login

import (
	"bytes"
	"testing"

	"gitlab.com/gitlab-org/cli/pkg/iostreams"

	"github.com/google/shlex"
	"github.com/stretchr/testify/assert"
	"github.com/zalando/go-keyring"
	"gitlab.com/gitlab-org/cli/commands/cmdtest"
)

func TestMain(m *testing.M) {
	cmdtest.InitTest(m, "auth_login_test")
}

func Test_NewCmdLogin(t *testing.T) {
	tests := []struct {
		name     string
		cli      string
		stdin    string
		wants    LoginOptions
		stdinTTY bool
		wantsErr bool
		err      string
	}{
		{
			name:  "nontty, stdin",
			stdin: "abc123\n",
			cli:   "--stdin",
			wants: LoginOptions{
				Hostname: "gitlab.com",
				Token:    "abc123",
			},
		},
		{
			name:  "tty, stdin",
			stdin: "def456",
			cli:   "--stdin",
			wants: LoginOptions{
				Hostname: "gitlab.com",
				Token:    "def456",
			},
			stdinTTY: true,
		},
		{
			name:     "nontty, hostname",
			cli:      "--hostname salsa.debian.org",
			wantsErr: true,
			err:      "could not get login type",
			stdinTTY: false,
		},
		{
			name:     "nontty",
			cli:      "",
			wantsErr: true,
			err:      "could not prompt",
			stdinTTY: false,
		},
		{
			name:  "nontty, stdin, hostname",
			cli:   "--hostname db.org --stdin",
			stdin: "abc123\n",
			wants: LoginOptions{
				Hostname: "db.org",
				Token:    "abc123",
			},
		},
		{
			name:  "tty, stdin, hostname",
			stdin: "gli789",
			cli:   "--stdin --hostname gl.io",
			wants: LoginOptions{
				Hostname: "gl.io",
				Token:    "gli789",
			},
			stdinTTY: true,
		},
		{
			name: "non-interactive hostname, token, api-host",
			cli:  "--hostname gl.io --token foo --api-host api.gitlab.com",
			wants: LoginOptions{
				Hostname: "gl.io",
				Token:    "foo",
				ApiHost:  "api.gitlab.com",
			},
		},
		{
			name: "non-interactive hostname, token, api-host, api-protocol, git-protocol",
			cli:  "--hostname gl.io --token foo --api-host gl.io:3443 --api-protocol https --git-protocol ssh",
			wants: LoginOptions{
				Hostname:    "gl.io",
				Token:       "foo",
				ApiHost:     "gl.io:3443",
				ApiProtocol: "https",
				GitProtocol: "ssh",
			},
		},
		{
			name:  "non-interactive hostname, api-host, api-protocol, git-protocol with stdin token",
			cli:   "--hostname gl.io --api-host gl.io:3443 --api-protocol https --git-protocol ssh --stdin",
			stdin: "gli789",
			wants: LoginOptions{
				Hostname:    "gl.io",
				Token:       "gli789",
				ApiHost:     "gl.io:3443",
				ApiProtocol: "https",
				GitProtocol: "ssh",
			},
			stdinTTY: true,
		},
		{
			name:     "api-host in interactive mode",
			cli:      "--hostname gl.io --api-host api.gitlab.com",
			wantsErr: true,
			err:      "api-host, api-protocol, and git-protocol can only be used in non-interactive mode",
		},
		{
			name:     "api-protocol in interactive mode",
			cli:      "--hostname gl.io --api-protocol http",
			wantsErr: true,
			err:      "api-host, api-protocol, and git-protocol can only be used in non-interactive mode",
		},
		{
			name:     "git-protocol in interactive mode",
			cli:      "--hostname gl.io --git-protocol ssh",
			wantsErr: true,
			err:      "api-host, api-protocol, and git-protocol can only be used in non-interactive mode",
		},
		// TODO: how to test survey
		//{
		//	name:     "tty, hostname",
		//	cli:      "--hostname local.dev",
		//	wants: LoginOptions{
		//		Hostname:    "local.dev",
		//		Token:       "",
		//		Interactive: true,
		//	},
		//	stdinTTY: true,
		//},
		//{
		//	name:     "tty",
		//	cli:      "",
		//	wants: LoginOptions{
		//		Hostname:    "",
		//		Token:       "",
		//		Interactive: true,
		//	},
		//	stdinTTY: true,
		//},
		{
			name:     "token and stdin",
			cli:      "--token xxxx --stdin",
			wantsErr: true,
			err:      "specify one of '--token' or '--stdin'. You cannot use both flags at the same time",
		},
		{
			name: "no keyring, token",
			cli:  "--token glpat-123",
			wants: LoginOptions{
				Hostname:   "gitlab.com",
				Token:      "glpat-123",
				UseKeyring: false,
			},
		},
		{
			name: "keyring, token",
			cli:  "--token glpat-123 --use-keyring",
			wants: LoginOptions{
				Hostname:   "gitlab.com",
				Token:      "glpat-123",
				UseKeyring: true,
			},
		},
	}

	// Enable keyring mocking, so no changes are made to it accidentally and to prevent failing in some environments
	keyring.MockInit()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := t.TempDir()
			t.Setenv("GLAB_CONFIG_DIR", d)

			io, stdin, _, _ := iostreams.Test()
			f := cmdtest.StubFactory("https://gitlab.com/cli-automated-testing/test")

			f.IO = io
			io.IsaTTY = true
			io.IsErrTTY = true
			io.IsInTTY = tt.stdinTTY

			if tt.stdin != "" {
				stdin.WriteString(tt.stdin)
			}

			argv, err := shlex.Split(tt.cli)
			assert.NoError(t, err)

			cmd := NewCmdLogin(f)
			// TODO cobra hack-around
			cmd.Flags().BoolP("help", "x", false, "")

			cmd.SetArgs(argv)
			cmd.SetIn(&bytes.Buffer{})
			cmd.SetOut(&bytes.Buffer{})
			cmd.SetErr(&bytes.Buffer{})

			_, err = cmd.ExecuteC()

			if tt.wantsErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.err)
				return
			}
			assert.NoError(t, err)

			assert.Equal(t, tt.wants.Token, opts.Token)
			assert.Equal(t, tt.wants.Hostname, opts.Hostname)
			assert.Equal(t, tt.wants.Interactive, opts.Interactive)
			assert.Equal(t, tt.wants.ApiHost, opts.ApiHost)
			assert.Equal(t, tt.wants.ApiProtocol, opts.ApiProtocol)
			assert.Equal(t, tt.wants.GitProtocol, opts.GitProtocol)
		})
	}
}

func Test_hostnameValidator(t *testing.T) {
	testMap := make(map[string]string)
	testMap["profclems"] = "glab"

	testCases := []struct {
		name     string
		hostname interface{}
		expected string
	}{
		{
			name:     "valid",
			hostname: "localhost",
		},
		{
			name:     "valid-default-value",
			hostname: "gitlab.com",
		},
		{
			name:     "valid-external-instance-alpine",
			hostname: "gitlab.alpinelinux.org",
		},
		{
			name:     "valid-external-instance-freedesktop",
			hostname: "gitlab.freedesktop.org",
		},
		{
			name:     "valid-external-instance-gnome",
			hostname: "gitlab.gnome.org",
		},
		{
			name:     "valid-external-instance-debian",
			hostname: "salsa.debian.org",
		},
		{
			name:     "valid-external-instance-ip",
			hostname: "1.1.1.1",
		},
		{
			name:     "valid-external-instance-ip-with-port",
			hostname: "1.1.1.1:8080",
		},
		{
			name:     "empty",
			hostname: "",
			expected: "a value is required.",
		},
		{
			name:     "valid-hostname-slash",
			hostname: "localhost:9999/host",
		},
		{
			name:     "hostname-with-valid-port",
			hostname: "gitlab.mycompany.com:4000",
		},
		{
			name:     "hostname-with-invalid-port",
			hostname: "local:host",
			expected: `invalid hostname "local:host"`,
		},
		{
			name:     "valid-with-int-type",
			hostname: 10,
		},
		{
			name:     "valid-with-slice-string-type",
			hostname: []string{"local", "host"},
			expected: `invalid hostname "[local host]"`,
		},
		{
			name:     "invalid-with-map-type",
			hostname: testMap,
			expected: `invalid hostname "map[profclems:glab]"`,
		},
	}
	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			err := hostnameValidator(tC.hostname)
			if tC.expected == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tC.expected)
			}
		})
	}
}

func Test_keyringLogin(t *testing.T) {
	keyring.MockInit()

	token, err := keyring.Get("glab:gitlab.com", "")
	assert.Error(t, err)
	assert.Equal(t, "", token)

	f := cmdtest.StubFactory("https://gitlab.com/cli-automated-testing/test")
	cmd := NewCmdLogin(f)
	cmd.Flags().BoolP("help", "x", false, "")

	cmd.SetArgs([]string{"--use-keyring", "--token", "glpat-1234"})

	_, err = cmd.ExecuteC()
	assert.Nil(t, err)

	token, err = keyring.Get("glab:gitlab.com", "")
	assert.NoError(t, err)
	assert.Equal(t, "glpat-1234", token)
}
