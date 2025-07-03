package merge

import (
	"net/http"
	"testing"

	"github.com/google/shlex"
	"gitlab.com/gitlab-org/cli/internal/testing/cmdtest"

	"github.com/MakeNowJust/heredoc/v2"

	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/cli/internal/testing/httpmock"
	"gitlab.com/gitlab-org/cli/test"
)

func runCommand(t *testing.T, rt http.RoundTripper, cli string) (*test.CmdOut, error) {
	ios, _, stdout, stderr := cmdtest.TestIOStreams()

	factory := cmdtest.NewTestFactory(ios,
		cmdtest.WithGitLabClient(cmdtest.NewTestApiClient(t, &http.Client{Transport: rt}, "", "").Lab()),
		cmdtest.WithBaseRepo("OWNER", "REPO"),
	)

	cmd := NewCmdMerge(factory)

	argv, err := shlex.Split(cli)
	if err != nil {
		return nil, err
	}
	cmd.SetArgs(argv)

	_, err = cmd.ExecuteC()
	return &test.CmdOut{
		OutBuf: stdout,
		ErrBuf: stderr,
	}, err
}

func TestMrApprove(t *testing.T) {
	fakeHTTP := httpmock.New()
	defer fakeHTTP.Verify(t)

	fakeHTTP.RegisterResponder(http.MethodGet, `/projects/OWNER/REPO/merge_requests/123`,
		httpmock.NewFileResponse(http.StatusOK, "./testdata/mergeableMr.json"))

	fakeHTTP.RegisterResponder(http.MethodPut, `/projects/OWNER/REPO/merge_requests/123/merge`,
		httpmock.NewFileResponse(http.StatusOK, "./testdata/mergedMr.json"))

	mrID := "123"
	output, err := runCommand(t, fakeHTTP, mrID)
	if assert.NoErrorf(t, err, "error running command `mr merge %s`", mrID) {
		out := output.String()

		assert.Equal(t, heredoc.Doc(`
		✓ Pipeline succeeded.
		✓ Merged!
		https://gitlab.com/OWNER/REPO/-/merge_requests/123
		`), out)
		assert.Empty(t, output.Stderr())
	}
}
