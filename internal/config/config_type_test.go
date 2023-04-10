package config

import (
	"bytes"
	"testing"

	"github.com/MakeNowJust/heredoc"
	"github.com/stretchr/testify/assert"
	"github.com/zalando/go-keyring"
)

func Test_fileConfig_Set(t *testing.T) {
	mainBuf := bytes.Buffer{}
	aliasesBuf := bytes.Buffer{}
	defer StubWriteConfig(&mainBuf, &aliasesBuf)()

	c := NewBlankConfig()
	assert.NoError(t, c.Set("", "editor", "nano"))
	assert.NoError(t, c.Set("gitlab.com", "git_protocol", "ssh"))
	assert.NoError(t, c.Set("example.com", "editor", "vim"))
	assert.NoError(t, c.Set("gitlab.com", "username", "hubot"))
	assert.NoError(t, c.WriteAll())
	// a, _ := c.Aliases()
	// assert.NoError(t, a.Set("co", "mr checkout"))
	// assert.NoError(t, a.Write())

	expected := heredoc.Doc(`# What protocol to use when performing git operations. Supported values: ssh, https
git_protocol: ssh
# What editor glab should run when creating issues, merge requests, etc.  This is a global config that cannot be overridden by hostname.
editor: nano
# What browser glab should run when opening links. This is a global config that cannot be overridden by hostname.
browser:
# Set your desired markdown renderer style. Available options are [dark, light, notty] or set a custom style. Refer to https://github.com/charmbracelet/glamour#styles
glamour_style: dark
# Allow glab to automatically check for updates and notify you when there are new updates
check_update: false
# Whether or not to display hyperlink escapes when listing things like issues or MRs
display_hyperlinks: false
# configuration specific for gitlab instances
hosts:
    gitlab.com:
        # What protocol to use to access the api endpoint. Supported values: http, https
        api_protocol: https
        # Configure host for api endpoint, defaults to the host itself
        api_host: gitlab.com
        # Your GitLab access token. Get an access token at https://gitlab.com/-/profile/personal_access_tokens
        token:
        git_protocol: ssh
        username: hubot
    example.com:
        editor: vim
# Default GitLab hostname to use
host: gitlab.com
`)
	assert.Equal(t, expected, mainBuf.String())
	assert.Equal(t, `ci: pipeline ci
co: mr checkout
`, aliasesBuf.String())
}

func Test_defaultConfig(t *testing.T) {
	mainBuf := bytes.Buffer{}
	hostsBuf := bytes.Buffer{}
	defer StubWriteConfig(&mainBuf, &hostsBuf)()

	cfg := NewBlankConfig()
	assert.NoError(t, cfg.Write())

	expected := heredoc.Doc(`# What protocol to use when performing git operations. Supported values: ssh, https
git_protocol: ssh
# What editor glab should run when creating issues, merge requests, etc.  This is a global config that cannot be overridden by hostname.
editor:
# What browser glab should run when opening links. This is a global config that cannot be overridden by hostname.
browser:
# Set your desired markdown renderer style. Available options are [dark, light, notty] or set a custom style. Refer to https://github.com/charmbracelet/glamour#styles
glamour_style: dark
# Allow glab to automatically check for updates and notify you when there are new updates
check_update: false
# Whether or not to display hyperlink escapes when listing things like issues or MRs
display_hyperlinks: false
# configuration specific for gitlab instances
hosts:
    gitlab.com:
        # What protocol to use to access the api endpoint. Supported values: http, https
        api_protocol: https
        # Configure host for api endpoint, defaults to the host itself
        api_host: gitlab.com
        # Your GitLab access token. Get an access token at https://gitlab.com/-/profile/personal_access_tokens
        token:
# Default GitLab hostname to use
host: gitlab.com
`)
	assert.Equal(t, expected, mainBuf.String())
	assert.Equal(t, "", hostsBuf.String())

	proto, err := cfg.Get("", "git_protocol")
	assert.Nil(t, err)
	assert.Equal(t, "ssh", proto)

	editor, err := cfg.Get("", "editor")
	assert.Nil(t, err)
	assert.Equal(t, "", editor)

	aliases, err := cfg.Aliases()
	assert.Nil(t, err)
	assert.Equal(t, len(aliases.All()), 2)
	expansion, _ := aliases.Get("co")
	assert.Equal(t, expansion, "mr checkout")
}

func Test_getFromKeyring(t *testing.T) {
	c := NewBlankConfig()

	err := c.Set("gitlab.com", "api_host", "gitlab.com")
	assert.NoError(t, err)

	keyring.MockInit()
	token, err := c.Get("gitlab.com", "token")
	assert.Nil(t, err)
	assert.Equal(t, token, "")

	err = keyring.Set("glab:gitlab.com", "", "glpat-1234")
	assert.NoError(t, err)

	token, err = c.Get("gitlab.com", "token")

	assert.Nil(t, err)
	assert.Equal(t, token, "glpat-1234")
}
