package list

import (
	"testing"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/acarl005/stripansi"
	"github.com/stretchr/testify/assert"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/internal/testing/cmdtest"
)

func Test_ScheduleList(t *testing.T) {
	io, _, stdout, stderr := cmdtest.TestIOStreams(cmdtest.WithTestIOStreamsAsTTY(true))
	f := cmdtest.NewTestFactory(io, cmdtest.WithConfig(config.NewFromString(heredoc.Doc(`
		hosts:
		  gitlab.com:
		    username: monalisa
		    token: OTOKEN
	`))))

	getSchedules = func(client *gitlab.Client, l *gitlab.ListPipelineSchedulesOptions, repo string) ([]*gitlab.PipelineSchedule, error) {
		_, err := f.BaseRepo()
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

	cmd := NewCmdList(f)
	cmdutils.EnableRepoOverride(cmd, f)

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
	io, _, stdout, stderr := cmdtest.TestIOStreams(cmdtest.WithTestIOStreamsAsTTY(true))
	stubFactory := cmdtest.NewTestFactory(io, cmdtest.WithConfig(config.NewFromString(heredoc.Doc(`
		hosts:
		  gitlab.com:
		    username: monalisa
		    token: OTOKEN
	`))))

	getSchedules = func(client *gitlab.Client, l *gitlab.ListPipelineSchedulesOptions, repo string) ([]*gitlab.PipelineSchedule, error) {
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
