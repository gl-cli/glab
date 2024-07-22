package create

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xanzy/go-gitlab"
	"gitlab.com/gitlab-org/cli/commands/cmdtest"
	"gitlab.com/gitlab-org/cli/pkg/httpmock"
	"gitlab.com/gitlab-org/cli/test"
)

func TestGenerateIssueWebURL(t *testing.T) {
	opts := &CreateOpts{
		Labels:         []string{"backend", "frontend"},
		Assignees:      []string{"johndoe", "janedoe"},
		Milestone:      15,
		Weight:         3,
		IsConfidential: true,
		BaseProject: &gitlab.Project{
			ID:     101,
			WebURL: "https://gitlab.example.com/gitlab-org/gitlab",
		},
		Title: "Autofill tests | for this @project",
	}

	u, err := generateIssueWebURL(opts)

	expectedUrl := "https://gitlab.example.com/gitlab-org/gitlab/-/issues/new?" +
		"issue%5Bdescription%5D=%0A%2Flabel+~%22backend%22+~%22frontend%22%0A%2Fassign+johndoe%2C+janedoe%0A%2Fmilestone+%2515%0A%2Fweight+3%0A%2Fconfidential&" +
		"issue%5Btitle%5D=Autofill+tests+%7C+for+this+%40project"

	assert.NoError(t, err)
	assert.Equal(t, expectedUrl, u)
}

func runCommand(rt http.RoundTripper, isTTY bool, cli string) (*test.CmdOut, error) {
	ios, _, stdout, stderr := cmdtest.InitIOStreams(isTTY, "")

	factory := cmdtest.InitFactory(ios, rt)

	_, _ = factory.HttpClient()

	cmd := NewCmdCreate(factory)

	return cmdtest.ExecuteCommand(cmd, cli, stdout, stderr)
}

func TestIssueCreateWhenIssuesDisabled(t *testing.T) {
	fakeHTTP := &httpmock.Mocker{
		MatchURL: httpmock.PathAndQuerystring,
	}
	defer fakeHTTP.Verify(t)

	fakeHTTP.RegisterResponder(http.MethodGet, "/api/v4/projects/OWNER/REPO?license=true&with_custom_attributes=true",
		httpmock.NewStringResponse(http.StatusOK, `{
							  "id": 37777023,
							  "description": "this is a test description",
							  "name": "REPO",
							  "name_with_namespace": "Test User / REPO",
							  "path": "REPO",
							  "path_with_namespace": "OWNER/REPO",
							  "created_at": "2022-07-13T02:04:56.151Z",
							  "default_branch": "main",
							  "http_url_to_repo": "https://gitlab.com/OWNER/REPO.git",
							  "web_url": "https://gitlab.com/OWNER/REPO",
							  "readme_url": "https://gitlab.com/OWNER/REPO/-/blob/main/README.md",
							  "issues_enabled": false
							}`))

	cli := `--title "test title" --description "test description"`

	output, err := runCommand(fakeHTTP, false, cli)
	assert.NotNil(t, err)
	assert.Empty(t, output.String())
	assert.Equal(t, "Issues are disabled for project \"OWNER/REPO\" or require project membership. "+
		"Make sure issues are enabled for the \"OWNER/REPO\" project, and if required, you are a member of the project.\n",
		output.Stderr())
}
