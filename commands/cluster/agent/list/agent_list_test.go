package list

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/MakeNowJust/heredoc/v2"
	"gitlab.com/gitlab-org/cli/pkg/httpmock"
	"gitlab.com/gitlab-org/cli/test"

	"gitlab.com/gitlab-org/cli/commands/cmdtest"

	"github.com/stretchr/testify/assert"
)

func runCommand(rt http.RoundTripper, isTTY bool, cli string, doHyperlinks string) (*test.CmdOut, error) {
	ios, _, stdout, stderr := cmdtest.InitIOStreams(isTTY, doHyperlinks)
	f := cmdtest.InitFactory(ios, rt)

	// Note: This sets the RoundTripper, which is necessary for stubs to work.
	_, _ = f.HttpClient()

	cmd := NewCmdAgentList(f)

	return cmdtest.ExecuteCommand(cmd, cli, stdout, stderr)
}

func TestAgentList(t *testing.T) {
	fakeHTTP := httpmock.New()
	defer fakeHTTP.Verify(t)

	deterministicCreatedAt := time.Now().Add(-24 * time.Hour).Format(time.RFC3339)
	fakeHTTP.RegisterResponder(http.MethodGet, "/projects/OWNER/REPO/cluster_agents",
		httpmock.NewStringResponse(http.StatusOK, fmt.Sprintf(`
			[
			  {
				"id": 1,
				"name": "local",
				"created_at": "%[1]s"
			  },
			  {
				"id": 2,
				"name": "prd",
				"created_at": "%[1]s"
			  }
			]
	`, deterministicCreatedAt)))

	output, err := runCommand(fakeHTTP, true, "", "")
	if err != nil {
		t.Errorf("error running command `cluster agent list`: %v", err)
	}

	assert.Equal(t, heredoc.Doc(`
		Showing 2 agents on OWNER/REPO. (Page 1)

		ID	Name	Created At
		1	local	about 1 day ago
		2	prd	about 1 day ago

	`), output.String())
	assert.Equal(t, ``, output.Stderr())
}

func TestAgentList_Pagination(t *testing.T) {
	fakeHTTP := httpmock.New()
	defer fakeHTTP.Verify(t)

	deterministicCreatedAt := time.Now().Add(-24 * time.Hour).Format(time.RFC3339)
	fakeHTTP.RegisterResponder(http.MethodGet, "/projects/OWNER/REPO/cluster_agents",
		httpmock.NewStringResponse(http.StatusOK, fmt.Sprintf(`
			[
			  {
				"id": 1,
				"name": "local",
				"created_at": "%[1]s"
			  },
			  {
				"id": 2,
				"name": "prd",
				"created_at": "%[1]s"
			  }
			]
		`, deterministicCreatedAt)))

	cli := "--page 42 --per-page 10"
	output, err := runCommand(fakeHTTP, true, cli, "")
	if err != nil {
		t.Errorf("error running command `cluster agent list`: %v", err)
	}

	assert.Equal(t, heredoc.Doc(`
		Showing 2 agents on OWNER/REPO. (Page 42)

		ID	Name	Created At
		1	local	about 1 day ago
		2	prd	about 1 day ago

	`), output.String())
	assert.Equal(t, ``, output.Stderr())
}
