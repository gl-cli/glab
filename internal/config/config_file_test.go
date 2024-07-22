package config

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cli/test"
	"gopkg.in/yaml.v3"
)

func eq(t *testing.T, got interface{}, expected interface{}) {
	t.Helper()
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("expected: %v, got: %v", expected, got)
	}
}

func Test_parseConfig(t *testing.T) {
	defer StubConfig(`---
hosts:
  gitlab.com:
    username: monalisa
    token: OTOKEN
aliases:
`, "")()
	test.ClearEnvironmentVariables(t)

	config, err := ParseConfig("config.yml")
	eq(t, err, nil)
	username, err := config.Get("gitlab.com", "username")
	eq(t, err, nil)
	eq(t, username, "monalisa")
	token, err := config.Get("gitlab.com", "token")
	eq(t, err, nil)
	eq(t, token, "OTOKEN")
}

func Test_parseConfig_multipleHosts(t *testing.T) {
	defer StubConfig(`---
hosts:
  gitlab.example.com:
    username: wrongusername
    token: NOTTHIS
  gitlab.com:
    username: monalisa
    token: OTOKEN
`, "")()
	test.ClearEnvironmentVariables(t)

	config, err := ParseConfig("config.yml")
	eq(t, err, nil)
	username, err := config.Get("gitlab.com", "username")
	eq(t, err, nil)
	eq(t, username, "monalisa")
	token, err := config.Get("gitlab.com", "token")
	eq(t, err, nil)
	eq(t, token, "OTOKEN")
}

func Test_parseConfig_Hosts(t *testing.T) {
	defer StubConfig(`---
hosts:
  gitlab.com:
    username: monalisa
    token: OTOKEN
`, `
`)()
	test.ClearEnvironmentVariables(t)

	config, err := ParseConfig("config.yml")
	eq(t, err, nil)
	username, err := config.Get("gitlab.com", "username")
	eq(t, err, nil)
	eq(t, username, "monalisa")
	token, err := config.Get("gitlab.com", "token")
	eq(t, err, nil)
	eq(t, token, "OTOKEN")
}

func Test_parseConfig_Local(t *testing.T) {
	test.ClearEnvironmentVariables(t)

	defer StubConfig(`---
git_protocol: ssh
editor: vim
local:
  git_protocol: https
  editor: nano
`, `
`)()
	config, err := ParseConfig("config.yml")
	eq(t, err, nil)
	gitProtocol, err := config.Get("", "git_protocol")
	eq(t, err, nil)
	eq(t, gitProtocol, "https")
	editor, err := config.Get("", "editor")
	eq(t, err, nil)
	eq(t, editor, "nano")
}

func Test_Get_configReadSequence(t *testing.T) {
	test.ClearEnvironmentVariables(t)

	defer StubConfig(`---
git_protocol: ssh
editor: vim
browser: mozilla
local:
  git_protocol: https
  editor:
  browser: chrome
`, `
`)()
	t.Setenv("BROWSER", "opera")

	config, err := ParseConfig("config.yml")
	eq(t, err, nil)
	gitProtocol, err := config.Get("", "git_protocol")
	eq(t, err, nil)
	eq(t, gitProtocol, "https")
	token, err := config.Get("", "editor")
	eq(t, err, nil)
	eq(t, token, "vim")
	browser, err := config.Get("", "browser")
	eq(t, err, nil)
	eq(t, browser, "opera")
	l, _ := config.Local()
	t.Log(l.All())
}

func Test_parseConfig_AliasesFile(t *testing.T) {
	defer StubConfig("", `---
ci: pipeline ci
co: mr checkout
`)()
	config, err := ParseConfig("aliases.yml")
	eq(t, err, nil)
	aliases, err := config.Aliases()
	eq(t, err, nil)
	a, isAlias := aliases.Get("ci")
	eq(t, isAlias, true)
	eq(t, a, "pipeline ci")
	b, isAlias := aliases.Get("co")
	eq(t, isAlias, true)
	eq(t, b, "mr checkout")
	eq(t, len(aliases.All()), 2)
}

func Test_parseConfig_hostFallback(t *testing.T) {
	defer StubConfig(`---
git_protocol: ssh
hosts:
  gitlab.com:
    username: monalisa
    token: OTOKEN
  gitlab.example.com:
    username: wrongusername
    token: NOTTHIS
    git_protocol: https
`, `
`)()
	config, err := ParseConfig("config.yml")
	eq(t, err, nil)
	val, err := config.Get("gitlab.example.com", "git_protocol")
	eq(t, err, nil)
	eq(t, val, "https")
	val, err = config.Get("gitlab.com", "git_protocol")
	eq(t, err, nil)
	eq(t, val, "ssh")
	val, err = config.Get("nonexist.io", "git_protocol")
	eq(t, err, nil)
	eq(t, val, "ssh")
}

func Test_parseConfigFile(t *testing.T) {
	tests := []struct {
		contents string
		wantsErr bool
	}{
		{
			contents: "",
			wantsErr: true,
		},
		{
			contents: " ",
			wantsErr: false,
		},
		{
			contents: "\n",
			wantsErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("contents: %q", tt.contents), func(t *testing.T) {
			defer StubConfig(tt.contents, "")()
			_, yamlRoot, err := ParseConfigFile("config.yml")
			if tt.wantsErr != (err != nil) {
				t.Fatalf("got error: %v", err)
			}
			if tt.wantsErr {
				return
			}
			assert.Equal(t, yaml.MappingNode, yamlRoot.Content[0].Kind)
			assert.Equal(t, 0, len(yamlRoot.Content[0].Content))
		})
	}
}

func Test_ParseConfigFilePermissions(t *testing.T) {
	tests := map[string]struct {
		permissions int
		wantErr     bool
	}{
		"bad permissions": {
			permissions: 0o755,
			wantErr:     true,
		},
		"normal permissions": {
			permissions: 0o600,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tempDir := t.TempDir()
			configFile := filepath.Join(tempDir, "config.yml")

			err := os.WriteFile(
				configFile,
				[]byte("---\nhost: https://gitlab.mycompany.global"),
				os.FileMode(tt.permissions),
			)
			require.NoError(t, err)

			_, err = ParseConfig(configFile)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func Test_parseConfigHostEnv(t *testing.T) {
	t.Setenv("GITLAB_URI", "https://gitlab.mycompany.env")

	defer StubConfig(`---
host: https://gitlab.mycompany.global
local:
  host: https://gitlab.mycompany.local
`, `
`)()
	config, err := ParseConfig("config.yml")
	eq(t, err, nil)

	val, err := config.Get("", "host")
	eq(t, err, nil)
	eq(t, val, "https://gitlab.mycompany.env")
}

func Test_parseConfigHostLocal(t *testing.T) {
	defer StubConfig(`---
host: https://gitlab.mycompany.global
local:
  host: https://gitlab.mycompany.local
`, `
`)()
	config, err := ParseConfig("config.yml")
	eq(t, err, nil)

	val, err := config.Get("", "host")
	eq(t, err, nil)
	eq(t, val, "https://gitlab.mycompany.local")
}

func Test_parseConfigHostGlobal(t *testing.T) {
	defer StubConfig(`---
host: https://gitlab.mycompany.org
`, `
`)()
	config, err := ParseConfig("config.yml")
	eq(t, err, nil)

	val, err := config.Get("", "host")
	eq(t, err, nil)
	eq(t, val, "https://gitlab.mycompany.org")
}
