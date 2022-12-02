package list

import (
	"net/http"
	"testing"

	"gitlab.com/gitlab-org/cli/pkg/iostreams"

	"github.com/MakeNowJust/heredoc"

	"github.com/alecthomas/assert"
	"github.com/xanzy/go-gitlab"
	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/pkg/httpmock"
	"gitlab.com/gitlab-org/cli/test"
)

func runCommand(rt http.RoundTripper) (*test.CmdOut, error) {
	ios, _, stdout, stderr := iostreams.Test()

	factory := &cmdutils.Factory{
		IO: ios,
		HttpClient: func() (*gitlab.Client, error) {
			a, err := api.TestClient(&http.Client{Transport: rt}, "", "", false)
			if err != nil {
				return nil, err
			}
			return a.Lab(), err
		},
		BaseRepo: func() (glrepo.Interface, error) {
			return glrepo.New("OWNER", "REPO"), nil
		},
	}

	// TODO: shouldn't be there but the stub doesn't work without it
	_, _ = factory.HttpClient()

	cmd := NewCmdList(factory)

	_, err := cmd.ExecuteC()
	return &test.CmdOut{
		OutBuf: stdout,
		ErrBuf: stderr,
	}, err
}

func TestLabelList(t *testing.T) {
	fakeHTTP := &httpmock.Mocker{}
	defer fakeHTTP.Verify(t)

	fakeHTTP.RegisterResponder("GET", "/api/v4/projects/OWNER/REPO/labels",
		httpmock.NewStringResponse(200, `
	[
		{
			"id":1,
			"name":"bug",
			"description":null,
			"text_color":"#FFFFFF",
			"color":"#6699cc",
			"priority":null,
			"is_project_label":true
		},
		{
			"id":2,
			"name":"ux",
			"description":null,
			"text_color":"#FFFFFF",
			"color":"#3cb371",
			"priority":null,
			"is_project_label":true
		}
	]
	`))

	output, err := runCommand(fakeHTTP)
	if err != nil {
		t.Errorf("error running command `label list`: %v", err)
	}

	out := output.String()

	assert.Equal(t, heredoc.Doc(`
		Showing label 2 of 2 on OWNER/REPO

		 bug
		 ux
 
	`), out)
	assert.Empty(t, output.Stderr())
}