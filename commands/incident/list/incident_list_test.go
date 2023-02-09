package list

import (
	"net/http"
	"regexp"
	"testing"

	"github.com/MakeNowJust/heredoc"
	"github.com/alecthomas/assert"
	"github.com/xanzy/go-gitlab"
	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/commands/cmdtest"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	issuableListCmd "gitlab.com/gitlab-org/cli/commands/issuable/list"
	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/pkg/httpmock"
	"gitlab.com/gitlab-org/cli/pkg/iostreams"
	"gitlab.com/gitlab-org/cli/test"
)

func runCommand(rt http.RoundTripper, isTTY bool, cli string, runE func(opts *issuableListCmd.ListOptions) error, doHyperlinks string) (*test.CmdOut, error) {
	ios, _, stdout, stderr := cmdtest.InitIOStreams(isTTY, doHyperlinks)
	factory := cmdtest.InitFactory(ios, rt)

	// TODO: shouldn't be there but the stub doesn't work without it
	_, _ = factory.HttpClient()

	cmd := NewCmdList(factory, runE)

	return cmdtest.ExecuteCommand(cmd, cli, stdout, stderr)
}

func TestNewCmdList(t *testing.T) {
	ios, _, _, _ := iostreams.Test()
	ios.IsaTTY = true
	ios.IsInTTY = true
	ios.IsErrTTY = true

	fakeHTTP := httpmock.New()
	defer fakeHTTP.Verify(t)

	factory := &cmdutils.Factory{
		IO: ios,
		HttpClient: func() (*gitlab.Client, error) {
			a, err := api.TestClient(&http.Client{Transport: fakeHTTP}, "", "", false)
			if err != nil {
				return nil, err
			}
			return a.Lab(), err
		},
		Config: func() (config.Config, error) {
			return config.NewBlankConfig(), nil
		},
		BaseRepo: func() (glrepo.Interface, error) {
			return glrepo.New("OWNER", "REPO"), nil
		},
	}
	t.Run("Incident_NewCmdList", func(t *testing.T) {
		gotOpts := &issuableListCmd.ListOptions{}
		err := NewCmdList(factory, func(opts *issuableListCmd.ListOptions) error {
			gotOpts = opts
			return nil
		}).Execute()

		assert.Nil(t, err)
		assert.Equal(t, factory.IO, gotOpts.IO)

		gotBaseRepo, _ := gotOpts.BaseRepo()
		expectedBaseRepo, _ := factory.BaseRepo()
		assert.Equal(t, gotBaseRepo, expectedBaseRepo)
	})
}

func TestIncidentList_tty(t *testing.T) {
	fakeHTTP := httpmock.New()
	defer fakeHTTP.Verify(t)

	fakeHTTP.RegisterResponder("GET", "/projects/OWNER/REPO/issues",
		httpmock.NewFileResponse(200, "./fixtures/incidentList.json"))

	output, err := runCommand(fakeHTTP, true, "", nil, "")
	if err != nil {
		t.Errorf("error running command `incident list`: %v", err)
	}

	out := output.String()
	timeRE := regexp.MustCompile(`\d+ years`)
	out = timeRE.ReplaceAllString(out, "X years")

	assert.Equal(t, heredoc.Doc(`
		Showing 1 open incident in OWNER/REPO that match your search (Page 1)

		#8	OWNER/REPO/issues/8	Incident	(foo, baz)	about X years ago

	`), out)
	assert.Equal(t, ``, output.Stderr())
}
