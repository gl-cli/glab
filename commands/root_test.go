package commands

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/cli/commands/cmdtest"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/pkg/iostreams"
	"gitlab.com/gitlab-org/cli/test"
)

func TestMain(m *testing.M) {
	cmdtest.InitTest(m, "")
}

func setupIOStreams() *iostreams.IOStreams {
	return iostreams.New(
		iostreams.WithStdin(os.Stdin, iostreams.IsTerminal(os.Stdin)),
		iostreams.WithStdout(iostreams.NewColorable(os.Stdout), iostreams.IsTerminal(os.Stdout)),
		iostreams.WithStderr(iostreams.NewColorable(os.Stderr), iostreams.IsTerminal(os.Stderr)),
		iostreams.WithPagerCommand(iostreams.PagerCommandFromEnv()),
	)
}

func TestRootVersion(t *testing.T) {
	old := os.Stdout // keep backup of the real stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	rootCmd := NewCmdRoot(cmdutils.NewFactory(setupIOStreams(), false, nil), "v1.0.0", "abcdefgh")
	assert.Nil(t, rootCmd.Flag("version").Value.Set("true"))
	assert.Nil(t, rootCmd.Execute())

	out := test.ReturnBuffer(old, r, w)

	assert.Equal(t, "glab 1.0.0 (abcdefgh)\n", out)
}

func TestRootNoArg(t *testing.T) {
	old := os.Stdout // keep backup of the real stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	rootCmd := NewCmdRoot(cmdutils.NewFactory(setupIOStreams(), false, nil), "v1.0.0", "abcdefgh")
	assert.Nil(t, rootCmd.Execute())

	out := test.ReturnBuffer(old, r, w)

	assert.Contains(t, out, "GLab is an open source GitLab CLI tool that brings GitLab to your command line.\n")
	assert.Contains(t, out, `USAGE
  glab <command> <subcommand> [flags]

CORE COMMANDS`)
}
