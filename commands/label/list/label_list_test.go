package list

import (
	"net/http"
	"testing"

	"gitlab.com/gitlab-org/cli/pkg/iostreams"

	"github.com/MakeNowJust/heredoc"

	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/cli/commands/cmdtest"
	"gitlab.com/gitlab-org/cli/pkg/httpmock"
	"gitlab.com/gitlab-org/cli/test"
)

func runCommand(rt http.RoundTripper) (*test.CmdOut, error) {
	ios, _, stdout, stderr := iostreams.Test()
	factory := cmdtest.InitFactory(ios, rt)

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

	fakeHTTP.RegisterResponder(http.MethodGet, "/api/v4/projects/OWNER/REPO/labels",
		httpmock.NewStringResponse(http.StatusOK, `
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
			"description":"User Experience",
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

		 bug (#6699cc)
		 ux -> User Experience (#3cb371)
 
	`), out)
	assert.Empty(t, output.Stderr())
}
