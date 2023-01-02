package list

import (
	"testing"

	"github.com/acarl005/stripansi"
	"github.com/stretchr/testify/assert"
	"github.com/xanzy/go-gitlab"
	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/commands/cmdtest"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/pkg/iostreams"
)

func Test_ScheduleList(t *testing.T) {
	defer config.StubConfig(`---
hosts:
  gitlab.com:
    username: monalisa
    token: OTOKEN
`, "")()

	io, _, stdout, stderr := iostreams.Test()
	stubFactory, _ := cmdtest.StubFactoryWithConfig("")
	stubFactory.IO = io
	stubFactory.IO.IsaTTY = true
	stubFactory.IO.IsErrTTY = true

	api.GetSchedules = func(client *gitlab.Client, l *gitlab.ListPipelineSchedulesOptions, repo string) ([]*gitlab.PipelineSchedule, error) {
		_, err := stubFactory.BaseRepo()
		if err != nil {
			return nil, err
		}

		return []*gitlab.PipelineSchedule{
			{
				ID:          1,
				Description: "foo",
				Cron:        "* * * * *",
				Owner: &gitlab.User{
					ID:       1,
					Username: "bar",
				},
				Active: true,
			},
		}, nil
	}

	cmd := NewCmdList(stubFactory)
	cmdutils.EnableRepoOverride(cmd, stubFactory)

	t.Run("Schedule exists", func(t *testing.T) {
		_, err := cmd.ExecuteC()
		if err != nil {
			t.Fatal(err)
		}

		out := stripansi.Strip(stdout.String())

		assert.Contains(t, out, "1\tfoo\t* * * * *\tbar\ttrue")
		assert.Equal(t, "", stderr.String())
	})
}

func Test_NoScheduleList(t *testing.T) {
	defer config.StubConfig(`---
hosts:
  gitlab.com:
    username: monalisa
    token: OTOKEN
`, "")()

	io, _, stdout, stderr := iostreams.Test()
	stubFactory, _ := cmdtest.StubFactoryWithConfig("")
	stubFactory.IO = io
	stubFactory.IO.IsaTTY = true
	stubFactory.IO.IsErrTTY = true

	api.GetSchedules = func(client *gitlab.Client, l *gitlab.ListPipelineSchedulesOptions, repo string) ([]*gitlab.PipelineSchedule, error) {
		_, err := stubFactory.BaseRepo()
		if err != nil {
			return nil, err
		}

		return nil, nil
	}

	cmd := NewCmdList(stubFactory)
	cmdutils.EnableRepoOverride(cmd, stubFactory)

	t.Run("No schedules exist", func(t *testing.T) {
		_, err := cmd.ExecuteC()
		if err != nil {
			t.Fatal(err)
		}

		out := stripansi.Strip(stdout.String())

		assert.Contains(t, out, "No schedules available on")
		assert.Equal(t, "", stderr.String())
	})
}
