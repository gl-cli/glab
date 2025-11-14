//go:build integration

package ask

import (
	"bytes"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/survivorbat/huhtest"

	"gitlab.com/gitlab-org/cli/internal/api"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/internal/iostreams"
	"gitlab.com/gitlab-org/cli/internal/testing/cmdtest"
	"gitlab.com/gitlab-org/cli/test"
)

func TestAskGit_Integration(t *testing.T) {
	glTestHost := test.GetHostOrSkip(t)
	t.Setenv("GITLAB_HOST", glTestHost)

	cfg, err := config.Init()
	require.NoError(t, err)

	rIn, wIn := io.Pipe()
	rOut, wOut := io.Pipe()
	rErr, wErr := io.Pipe()

	closer := func() {
		rIn.Close()
		wIn.Close()
		rOut.Close()
		wOut.Close()
		rErr.Close()
		wErr.Close()
	}

	ios := iostreams.New(
		iostreams.WithStdin(rIn, false),
		iostreams.WithStdout(wOut, false),
		iostreams.WithStderr(wErr, false),
	)

	f := cmdutils.NewFactory(ios, false, cfg, api.BuildInfo{})

	rstdin, rstdout, cancel := huhtest.NewResponder().
		// FIXME: there is a bug in huhtest (I've created https://github.com/survivorbat/huhtest/issues/2)
		// which leads to wrong answers when the Confirm has an affirmative default.
		// Therefore, we need to invert our actual answer. We want to say `No.`, but here due to that
		// bug need to affirm instead.
		AddConfirm(runCmdsQuestion, huhtest.ConfirmAffirm).
		Start(t, 1*time.Hour)

	defer cancel()
	defer closer()

	stdout := &bytes.Buffer{}
	mwOut := io.MultiWriter(rstdout, stdout)

	var wg sync.WaitGroup

	wg.Add(2)
	go func() {
		defer wg.Done()
		_, _ = io.Copy(wIn, rstdin)
	}()
	go func() {
		defer wg.Done()
		_, _ = io.Copy(mwOut, rOut)
	}()

	cmd := NewCmdAsk(f)
	cli := "--git how to create a branch"

	_, err = cmdtest.ExecuteCommand(cmd, cli, nil, nil)
	require.NoError(t, err)

	cancel()
	closer()

	wg.Wait()

	out := stdout.String()

	for _, msg := range []string{"Commands", "Explanation", "git checkout"} {
		assert.Contains(t, out, msg)
	}
}
