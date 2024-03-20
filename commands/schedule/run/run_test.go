package run

import (
	"os"
	"testing"

	"github.com/acarl005/stripansi"
	"github.com/google/shlex"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xanzy/go-gitlab"
	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/commands/cmdtest"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/pkg/iostreams"
	"gitlab.com/gitlab-org/cli/test"
)

func Test_ScheduleRun(t *testing.T) {
	defer config.StubConfig(`---
hosts:
  gitlab.com:
    username: monalisa
    token: OTOKEN
`, "")()

	io, _, Stderr, stderr := iostreams.Test()
	stubFactory, _ := cmdtest.StubFactoryWithConfig("")
	stubFactory.IO = io
	stubFactory.IO.IsaTTY = true
	stubFactory.IO.IsErrTTY = true

	api.RunSchedule = func(client *gitlab.Client, repo string, schedule int, opts ...gitlab.RequestOptionFunc) error {
		_, err := stubFactory.BaseRepo()
		if err != nil {
			return err
		}
		return nil
	}

	testCases := []struct {
		Name        string
		ExpectedMsg []string
		wantErr     bool
		cli         string
		wantStderr  string
	}{
		{
			Name:        "Schedule 1 started",
			ExpectedMsg: []string{"Started schedule with ID 1"},
			cli:         "1",
		},
	}

	cmd := NewCmdRun(stubFactory)
	cmdutils.EnableRepoOverride(cmd, stubFactory)

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			argv, err := shlex.Split(tc.cli)
			require.NoError(t, err)
			cmd.SetArgs(argv)

			_, err = cmd.ExecuteC()
			if err != nil {
				if tc.wantErr == true {
					if assert.Error(t, err) {
						assert.Equal(t, tc.wantStderr, err.Error())
					}
					return
				}
			}

			out := stripansi.Strip(Stderr.String())

			for _, msg := range tc.ExpectedMsg {
				assert.Contains(t, out, msg)
				assert.Equal(t, "", stderr.String())
			}
		})
	}
}

func Test_ScheduleRunNoID(t *testing.T) {
	old := os.Stderr // keep backup of the real Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	assert.Error(t, NewCmdRun(&cmdutils.Factory{}).Execute())

	out := test.ReturnBuffer(old, r, w)

	assert.Contains(t, out, "Error: accepts 1 arg(s), received 0\nUsage:\n  run <id> [flags]\n\nExamples:\nglab schedule run 1\n\n\nFlags:\n  -h, --help   help for run\n\n")
}
