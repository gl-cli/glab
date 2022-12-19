package commands

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/cli/commands/cmdtest"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
)

func TestMain(m *testing.M) {
	cmdtest.InitTest(m, "")
}

func TestRootVersion(t *testing.T) {
	old := os.Stdout // keep backup of the real stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	rootCmd := NewCmdRoot(cmdutils.NewFactory(), "v1.0.0", "2020-01-01")
	assert.Nil(t, rootCmd.Flag("version").Value.Set("true"))
	assert.Nil(t, rootCmd.Execute())

	outC := make(chan string)
	// copy the output in a separate goroutine so printing can't block indefinitely
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		outC <- buf.String()
	}()

	// back to normal state
	w.Close()
	os.Stdout = old // restoring the real stdout
	out := <-outC

	assert.Equal(t, "glab version 1.0.0 (2020-01-01)\n", out)
}

func TestRootNoArg(t *testing.T) {
	old := os.Stdout // keep backup of the real stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	rootCmd := NewCmdRoot(cmdutils.NewFactory(), "v1.0.0", "2020-01-01")
	assert.Nil(t, rootCmd.Execute())

	outC := make(chan string)
	// copy the output in a separate goroutine so printing can't block indefinitely
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		outC <- buf.String()
	}()

	// back to normal state
	w.Close()
	os.Stdout = old // restoring the real stdout
	out := <-outC
	assert.Contains(t, out, "GLab is an open source GitLab CLI tool that brings GitLab to your command line.\n")
	assert.Contains(t, out, `USAGE
  glab <command> <subcommand> [flags]

CORE COMMANDS`)
}
