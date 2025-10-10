package config

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zalando/go-keyring"
	"gopkg.in/yaml.v3"
)

func Test_configLock(t *testing.T) {
	defaultRoot := NewBlankRoot()
	cfg := NewConfig(defaultRoot)
	out, err := yaml.Marshal(defaultRoot)
	require.NoError(t, err)

	configLockPath := filepath.Join("config.yaml.lock")

	err = os.Chmod(configLockPath, 0o600)
	require.NoError(t, err)

	expected, yml, err := ParseConfigFile(configLockPath)
	require.NoError(t, err)
	assert.Equal(t, string(expected), string(out))

	lockCfg := NewConfig(yml)

	expectedHosts, err := cfg.Hosts()
	require.NoError(t, err)
	lockHosts, err := lockCfg.Hosts()
	require.NoError(t, err)
	assert.Equal(t, expectedHosts, lockHosts)

	expectedAliases, err := cfg.Aliases()
	require.NoError(t, err)
	lockAliases, err := lockCfg.Aliases()
	require.NoError(t, err)
	assert.Equal(t, expectedAliases.All(), lockAliases.All())
}

func Test_fileConfig_Set(t *testing.T) {
	defer StubConfig(`---
git_protocol: ssh
editor: vim
hosts:
  gitlab.com:
    token:
    git_protocol: https
    username: user
`, `
`)()

	mainBuf := bytes.Buffer{}
	aliasesBuf := bytes.Buffer{}
	defer StubWriteConfig(&mainBuf, &aliasesBuf)()

	c, err := ParseConfig("config.yml")
	require.NoError(t, err)

	assert.NoError(t, c.Set("", "editor", "nano"))
	assert.NoError(t, c.Set("gitlab.com", "git_protocol", "ssh"))
	assert.NoError(t, c.Set("example.com", "username", "testUser"))
	assert.NoError(t, c.Set("gitlab.com", "username", "hubot"))
	assert.NoError(t, c.WriteAll())

	expected := heredoc.Doc(`
git_protocol: ssh
editor: nano
hosts:
    gitlab.com:
        token:
        git_protocol: ssh
        username: hubot
    example.com:
        username: testUser
`)
	assert.Equal(t, expected, mainBuf.String())
}

func Test_fileConfig_Set_Empty_Removes(t *testing.T) {
	defer StubConfig(`---
git_protocol: ssh
editor: vim
hosts:
  gitlab.com:
    token: foobar
    git_protocol: https
    username: user
`, `
`)()

	mainBuf := bytes.Buffer{}
	aliasesBuf := bytes.Buffer{}
	defer StubWriteConfig(&mainBuf, &aliasesBuf)()

	c, err := ParseConfig("config.yml")
	require.NoError(t, err)

	assert.NoError(t, c.Set("", "editor", ""))
	assert.NoError(t, c.Set("gitlab.com", "token", ""))
	assert.NoError(t, c.WriteAll())

	expected := heredoc.Doc(`
git_protocol: ssh
hosts:
    gitlab.com:
        git_protocol: https
        username: user
`)
	assert.Equal(t, expected, mainBuf.String())
}

func Test_defaultConfig(t *testing.T) {
	mainBuf := bytes.Buffer{}
	hostsBuf := bytes.Buffer{}
	defer StubWriteConfig(&mainBuf, &hostsBuf)()

	cfg := NewBlankConfig()
	assert.NoError(t, cfg.Write())
	assert.Equal(t, "", hostsBuf.String())

	proto, err := cfg.Get("", "git_protocol")
	assert.Nil(t, err)
	assert.Equal(t, "ssh", proto)

	editor, err := cfg.Get("", "editor")
	assert.Nil(t, err)
	assert.Equal(t, os.Getenv("EDITOR"), editor)

	aliases, err := cfg.Aliases()
	assert.Nil(t, err)
	assert.Equal(t, len(aliases.All()), 2)
	expansion, _ := aliases.Get("co")
	assert.Equal(t, expansion, "mr checkout")
}

func Test_getFromKeyring(t *testing.T) {
	mainBuf := bytes.Buffer{}
	hostsBuf := bytes.Buffer{}
	defer StubWriteConfig(&mainBuf, &hostsBuf)()

	c := NewBlankConfig()

	// Ensure host exists and its token is empty
	err := c.Set("gitlab.com", "token", "")
	require.NoError(t, err)
	err = c.Write()
	require.NoError(t, err)

	keyring.MockInit()
	token, _, err := c.GetWithSource("gitlab.com", "token", false)
	assert.NoError(t, err)
	assert.Equal(t, "", token)

	err = keyring.Set("glab:gitlab.com", "", "glpat-1234")
	require.NoError(t, err)

	token, _, err = c.GetWithSource("gitlab.com", "token", false)

	assert.NoError(t, err)
	assert.Equal(t, "glpat-1234", token)
}

func Test_config_Get_NotFoundError(t *testing.T) {
	mainBuf := bytes.Buffer{}
	hostsBuf := bytes.Buffer{}
	defer StubWriteConfig(&mainBuf, &hostsBuf)()

	cfg := NewBlankConfig()

	local, err := cfg.Local()
	require.Nil(t, err)
	require.NotNil(t, local)

	_, err = local.FindEntry("git_protocol")
	require.Error(t, err)
	assert.True(t, isNotFoundError(err))
}

func TestCustomHeader_ResolvedValue_MissingEnvVar(t *testing.T) {
	// Ensure the environment variable doesn't exist
	os.Unsetenv("NONEXISTENT_VAR")

	header := CustomHeader{
		Name:         "X-Test-Header",
		ValueFromEnv: "NONEXISTENT_VAR",
	}

	value, err := header.ResolvedValue()
	require.Error(t, err)
	require.Empty(t, value)
	require.Contains(t, err.Error(), "environment variable \"NONEXISTENT_VAR\" for header \"X-Test-Header\" is not set or empty")
}

func TestCustomHeader_ResolvedValue_EmptyEnvVar(t *testing.T) {
	// Set environment variable to empty string
	t.Setenv("EMPTY_VAR", "")

	header := CustomHeader{
		Name:         "X-Test-Header",
		ValueFromEnv: "EMPTY_VAR",
	}

	value, err := header.ResolvedValue()
	require.Error(t, err)
	require.Empty(t, value)
	require.Contains(t, err.Error(), "environment variable \"EMPTY_VAR\" for header \"X-Test-Header\" is not set or empty")
}

func TestResolveCustomHeaders_MissingEnvVar(t *testing.T) {
	// Ensure the environment variable doesn't exist
	os.Unsetenv("MISSING_SECRET")

	configYAML := `
hosts:
  gitlab.com:
    custom_headers:
      - name: Cf-Access-Client-Secret
        valueFromEnv: MISSING_SECRET
`

	cfg := NewFromString(configYAML)
	headers, err := ResolveCustomHeaders(cfg, "gitlab.com")

	require.Error(t, err)
	require.Nil(t, headers)
	require.Contains(t, err.Error(), "failed to resolve header \"Cf-Access-Client-Secret\"")
	require.Contains(t, err.Error(), "environment variable \"MISSING_SECRET\" for header \"Cf-Access-Client-Secret\" is not set or empty")
}

func TestConfig_parseHosts_NoHosts(t *testing.T) {
	t.Parallel()

	cfg := &fileConfig{}
	// Create empty hosts node
	emptyHostsNode := &yaml.Node{Kind: yaml.MappingNode}

	_, err := cfg.parseHosts(emptyHostsNode)

	assert.True(t, isNotFoundError(err))
}
