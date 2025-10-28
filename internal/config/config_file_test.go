package config

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/adrg/xdg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cli/test"
	"gopkg.in/yaml.v3"
)

func eq(t *testing.T, got any, expected any) {
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

func Test_SearchConfigFile_UserConfig(t *testing.T) {
	test.ClearEnvironmentVariables(t)

	// Create a temporary user config directory
	userConfigDir := t.TempDir()
	userConfigFile := filepath.Join(userConfigDir, "glab-cli", "config.yml")

	// Create config file
	err := os.MkdirAll(filepath.Dir(userConfigFile), 0o750)
	require.NoError(t, err)
	err = os.WriteFile(userConfigFile, []byte("test: user"), 0o600)
	require.NoError(t, err)

	// Set XDG_CONFIG_HOME to temp directory
	t.Setenv("XDG_CONFIG_HOME", userConfigDir)
	xdg.Reload() // Reload XDG paths after env change

	// SearchConfigFile should find the user config
	foundPath, err := SearchConfigFile()
	require.NoError(t, err)
	assert.Equal(t, userConfigFile, foundPath)
}

func Test_SearchConfigFile_SystemConfig(t *testing.T) {
	test.ClearEnvironmentVariables(t)

	// Create temporary directories for system and user configs
	userConfigDir := t.TempDir()
	systemConfigDir := t.TempDir()
	systemConfigFile := filepath.Join(systemConfigDir, "glab-cli", "config.yml")

	// Create ONLY system config (no user config)
	err := os.MkdirAll(filepath.Dir(systemConfigFile), 0o750)
	require.NoError(t, err)
	err = os.WriteFile(systemConfigFile, []byte("test: system"), 0o600)
	require.NoError(t, err)

	// Set XDG environment variables
	t.Setenv("XDG_CONFIG_HOME", userConfigDir)
	t.Setenv("XDG_CONFIG_DIRS", systemConfigDir)
	xdg.Reload() // Reload XDG paths after env change

	// SearchConfigFile should find the system config
	foundPath, err := SearchConfigFile()
	require.NoError(t, err)
	assert.Equal(t, systemConfigFile, foundPath)
}

func Test_SearchConfigFile_Precedence(t *testing.T) {
	test.ClearEnvironmentVariables(t)

	// Create temporary directories
	userConfigDir := t.TempDir()
	systemConfigDir := t.TempDir()
	userConfigFile := filepath.Join(userConfigDir, "glab-cli", "config.yml")
	systemConfigFile := filepath.Join(systemConfigDir, "glab-cli", "config.yml")

	// Create both user and system configs
	err := os.MkdirAll(filepath.Dir(userConfigFile), 0o750)
	require.NoError(t, err)
	err = os.WriteFile(userConfigFile, []byte("test: user"), 0o600)
	require.NoError(t, err)

	err = os.MkdirAll(filepath.Dir(systemConfigFile), 0o750)
	require.NoError(t, err)
	err = os.WriteFile(systemConfigFile, []byte("test: system"), 0o600)
	require.NoError(t, err)

	// Set XDG environment variables
	t.Setenv("XDG_CONFIG_HOME", userConfigDir)
	t.Setenv("XDG_CONFIG_DIRS", systemConfigDir)
	xdg.Reload() // Reload XDG paths after env change

	// SearchConfigFile should prefer user config over system config
	foundPath, err := SearchConfigFile()
	require.NoError(t, err)
	assert.Equal(t, userConfigFile, foundPath)
}

func Test_SearchConfigFile_NotFound(t *testing.T) {
	test.ClearEnvironmentVariables(t)

	// Use empty temp directories (no config files)
	userConfigDir := t.TempDir()
	systemConfigDir := t.TempDir()

	t.Setenv("XDG_CONFIG_HOME", userConfigDir)
	t.Setenv("XDG_CONFIG_DIRS", systemConfigDir)
	xdg.Reload() // Reload XDG paths after env change

	// SearchConfigFile should return an error
	_, err := SearchConfigFile()
	require.Error(t, err)
}

func Test_SearchConfigFile_GLABConfigDirOverride(t *testing.T) {
	test.ClearEnvironmentVariables(t)

	// Create temp directories
	glabConfigDir := t.TempDir()
	userConfigDir := t.TempDir()
	glabConfigFile := filepath.Join(glabConfigDir, "config.yml")
	userConfigFile := filepath.Join(userConfigDir, "glab-cli", "config.yml")

	// Create both GLAB_CONFIG_DIR and user configs
	err := os.WriteFile(glabConfigFile, []byte("test: glab"), 0o600)
	require.NoError(t, err)

	err = os.MkdirAll(filepath.Dir(userConfigFile), 0o750)
	require.NoError(t, err)
	err = os.WriteFile(userConfigFile, []byte("test: user"), 0o600)
	require.NoError(t, err)

	// Set both environment variables
	t.Setenv("GLAB_CONFIG_DIR", glabConfigDir)
	t.Setenv("XDG_CONFIG_HOME", userConfigDir)

	// GLAB_CONFIG_DIR should take precedence
	foundPath, err := SearchConfigFile()
	require.NoError(t, err)
	assert.Equal(t, glabConfigFile, foundPath)
}

func Test_SearchConfigFile_GLABConfigDirNotFound(t *testing.T) {
	test.ClearEnvironmentVariables(t)

	// Use empty GLAB_CONFIG_DIR
	glabConfigDir := t.TempDir()
	t.Setenv("GLAB_CONFIG_DIR", glabConfigDir)

	// Should return os.ErrNotExist, not search XDG paths
	_, err := SearchConfigFile()
	require.ErrorIs(t, err, os.ErrNotExist)
}

func Test_ConfigFile_XDGCompliance(t *testing.T) {
	test.ClearEnvironmentVariables(t)

	userConfigDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", userConfigDir)
	xdg.Reload() // Reload XDG paths after env change

	configFile := ConfigFile()
	expectedPath := filepath.Join(userConfigDir, "glab-cli", "config.yml")

	assert.Equal(t, expectedPath, configFile)
}

func Test_ConfigDir_XDGCompliance(t *testing.T) {
	test.ClearEnvironmentVariables(t)

	userConfigDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", userConfigDir)
	xdg.Reload() // Reload XDG paths after env change

	configDir := ConfigDir()
	expectedPath := filepath.Join(userConfigDir, "glab-cli")

	assert.Equal(t, expectedPath, configDir)
}

func Test_ConfigFile_PureFunction(t *testing.T) {
	test.ClearEnvironmentVariables(t)

	// ConfigFile() should be a pure function that just returns a path
	userConfigDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", userConfigDir)
	xdg.Reload() // Reload XDG paths after env change

	configFile := ConfigFile()

	// Should always return a path without side effects
	assert.NotEmpty(t, configFile)
	assert.Contains(t, configFile, "glab-cli")
	assert.Contains(t, configFile, "config.yml")
	assert.True(t, filepath.IsAbs(configFile), "ConfigFile should return an absolute path")

	// The directory should NOT be created (pure function)
	configDir := filepath.Dir(configFile)
	_, err := os.Stat(configDir)
	assert.True(t, os.IsNotExist(err), "ConfigFile should not create directories")
}
