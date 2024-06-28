package delete

import (
	"net/http"
	"testing"
	"time"

	"github.com/spf13/pflag"
	"github.com/xanzy/go-gitlab"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/cli/pkg/iostreams"

	"github.com/MakeNowJust/heredoc/v2"

	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/cli/commands/cmdtest"
	"gitlab.com/gitlab-org/cli/pkg/httpmock"
	"gitlab.com/gitlab-org/cli/test"
)

func runCommand(rt http.RoundTripper, cli string) (*test.CmdOut, error) {
	ios, _, stdout, stderr := iostreams.Test()
	factory := cmdtest.InitFactory(ios, rt)

	_, _ = factory.HttpClient()

	cmd := NewCmdDelete(factory)

	return cmdtest.ExecuteCommand(cmd, cli, stdout, stderr)
}

func TestCIDelete(t *testing.T) {
	fakeHTTP := httpmock.New()
	defer fakeHTTP.Verify(t)

	fakeHTTP.RegisterResponder(http.MethodDelete, "/api/v4/projects/OWNER/REPO/pipelines/11111111",
		httpmock.NewStringResponse(http.StatusNoContent, ""),
	)

	pipelineId := "11111111"
	output, err := runCommand(fakeHTTP, pipelineId)
	if err != nil {
		t.Errorf("error running command `ci delete %s`: %v", pipelineId, err)
	}

	out := output.String()

	assert.Equal(t, heredoc.Doc(`
		✓ Pipeline #11111111 deleted successfully.
		`), out)
	assert.Empty(t, output.Stderr())
}

func TestCIDeleteNonExistingPipeline(t *testing.T) {
	fakeHTTP := httpmock.New()
	defer fakeHTTP.Verify(t)

	fakeHTTP.RegisterResponder(http.MethodDelete, "/api/v4/projects/OWNER/REPO/pipelines/11111111",
		httpmock.NewJSONResponse(http.StatusNotFound, "{message: 404 Not found}"),
	)

	pipelineId := "11111111"
	output, err := runCommand(fakeHTTP, pipelineId)

	require.Error(t, err)

	out := output.String()

	assert.Empty(t, out)
}

func TestCIDeleteWithWrongArgument(t *testing.T) {
	fakeHTTP := httpmock.New()
	defer fakeHTTP.Verify(t)

	pipelineId := "test"
	output, err := runCommand(fakeHTTP, pipelineId)

	require.Error(t, err)

	out := output.String()

	assert.Empty(t, out)
}

func TestCIDeleteByStatus(t *testing.T) {
	fakeHTTP := httpmock.New()
	fakeHTTP.MatchURL = httpmock.PathAndQuerystring
	defer fakeHTTP.Verify(t)

	fakeHTTP.RegisterResponder(http.MethodGet, "/api/v4/projects/OWNER/REPO/pipelines?status=success",
		httpmock.NewStringResponse(http.StatusOK, `
		[
			{
				"id": 11111111,
				"iid": 3,
				"project_id": 5,
				"sha": "c366255c71600e17519e802850ddcf7105d3cf66",
				"ref": "refs/merge-requests/1107/merge",
				"status": "success",
				"source": "merge_request_event",
				"created_at": "2020-12-01T01:15:50.559Z",
				"updated_at": "2020-12-01T01:36:41.737Z",
				"web_url": "https://gitlab.com/OWNER/REPO/-/pipelines/710046436"
			},
			{
				"id": 22222222,
				"iid": 4,
				"project_id": 5,
				"sha": "c9a7c0d9351cd1e71d1c2ad8277f3bc7e3c47d1f",
				"ref": "main",
				"status": "success",
				"source": "push",
				"created_at": "2020-11-30T18:20:47.571Z",
				"updated_at": "2020-11-30T18:39:40.092Z",
				"web_url": "https://gitlab.com/OWNER/REPO/-/pipelines/709793838"
			}
	]
	`))
	fakeHTTP.RegisterResponder(http.MethodDelete, "/api/v4/projects/OWNER/REPO/pipelines/11111111",
		httpmock.NewStringResponse(http.StatusNoContent, ""),
	)
	fakeHTTP.RegisterResponder(http.MethodDelete, "/api/v4/projects/OWNER/REPO/pipelines/22222222",
		httpmock.NewStringResponse(http.StatusNoContent, ""),
	)

	args := "--status=success"
	output, err := runCommand(fakeHTTP, args)
	require.NoError(t, err)

	out := output.String()

	assert.Equal(t, heredoc.Doc(`
		✓ Pipeline #11111111 deleted successfully.
		✓ Pipeline #22222222 deleted successfully.
		`), out)
	assert.Empty(t, output.Stderr())
}

func TestCIDeleteByStatusFailsWithArgument(t *testing.T) {
	fakeHTTP := httpmock.New()
	fakeHTTP.MatchURL = httpmock.PathAndQuerystring
	defer fakeHTTP.Verify(t)

	args := "--status=success 11111111"
	output, err := runCommand(fakeHTTP, args)
	assert.EqualError(t, err, "either a status filter or a pipeline ID must be passed, but not both.")

	assert.Empty(t, output.String())
	assert.Empty(t, output.Stderr())
}

func TestCIDeleteWithoutFilterFailsWithoutArgument(t *testing.T) {
	fakeHTTP := httpmock.New()
	fakeHTTP.MatchURL = httpmock.PathAndQuerystring
	defer fakeHTTP.Verify(t)

	pipelineId := ""
	output, err := runCommand(fakeHTTP, pipelineId)
	assert.EqualError(t, err, "accepts 1 arg(s), received 0")

	assert.Empty(t, output.String())
	assert.Empty(t, output.Stderr())
}

func TestCIDeleteMultiple(t *testing.T) {
	fakeHTTP := httpmock.New()
	defer fakeHTTP.Verify(t)

	fakeHTTP.RegisterResponder(http.MethodDelete, "/api/v4/projects/OWNER/REPO/pipelines/11111111",
		httpmock.NewStringResponse(http.StatusNoContent, ""),
	)
	fakeHTTP.RegisterResponder(http.MethodDelete, "/api/v4/projects/OWNER/REPO/pipelines/22222222",
		httpmock.NewStringResponse(http.StatusNoContent, ""),
	)

	pipelineId := "11111111,22222222"
	output, err := runCommand(fakeHTTP, pipelineId)
	if err != nil {
		t.Errorf("error running command `ci delete %s`: %v", pipelineId, err)
	}

	out := output.String()

	assert.Equal(t, heredoc.Doc(`
		✓ Pipeline #11111111 deleted successfully.
		✓ Pipeline #22222222 deleted successfully.
		`), out)
	assert.Empty(t, output.Stderr())
}

func TestCIDryRunDeleteNothing(t *testing.T) {
	fakeHTTP := httpmock.New()
	defer fakeHTTP.Verify(t)

	args := "--dry-run 11111111,22222222"
	output, err := runCommand(fakeHTTP, args)
	if err != nil {
		t.Errorf("error running command `ci delete %s`: %v", args, err)
	}

	out := output.String()

	assert.Equal(t, heredoc.Doc(`
		• Pipeline #11111111 will be deleted.
		• Pipeline #22222222 will be deleted.
		`), out)
	assert.Empty(t, output.Stderr())
}

func TestCIDeletedDryRunWithFilterDoesNotDelete(t *testing.T) {
	fakeHTTP := httpmock.New()
	fakeHTTP.MatchURL = httpmock.PathAndQuerystring
	defer fakeHTTP.Verify(t)

	fakeHTTP.RegisterResponder(http.MethodGet, "/api/v4/projects/OWNER/REPO/pipelines?status=success",
		httpmock.NewStringResponse(http.StatusOK, `
		[
			{
				"id": 11111111,
				"iid": 3,
				"project_id": 5,
				"sha": "c366255c71600e17519e802850ddcf7105d3cf66",
				"ref": "refs/merge-requests/1107/merge",
				"status": "success",
				"source": "merge_request_event",
				"created_at": "2020-12-01T01:15:50.559Z",
				"updated_at": "2020-12-01T01:36:41.737Z",
				"web_url": "https://gitlab.com/OWNER/REPO/-/pipelines/710046436"
			},
			{
				"id": 22222222,
				"iid": 4,
				"project_id": 5,
				"sha": "c9a7c0d9351cd1e71d1c2ad8277f3bc7e3c47d1f",
				"ref": "main",
				"status": "success",
				"source": "push",
				"created_at": "2020-11-30T18:20:47.571Z",
				"updated_at": "2020-11-30T18:39:40.092Z",
				"web_url": "https://gitlab.com/OWNER/REPO/-/pipelines/709793838"
			}
	]
	`))

	args := "--dry-run --status=success"
	output, err := runCommand(fakeHTTP, args)
	require.NoError(t, err)

	out := output.String()

	assert.Equal(t, heredoc.Doc(`
		• Pipeline #11111111 will be deleted.
		• Pipeline #22222222 will be deleted.
		`), out)
	assert.Empty(t, output.Stderr())
}

func TestCIDeleteBySource(t *testing.T) {
	fakeHTTP := httpmock.New()
	fakeHTTP.MatchURL = httpmock.PathAndQuerystring
	defer fakeHTTP.Verify(t)

	fakeHTTP.RegisterResponder(http.MethodGet, "/api/v4/projects/OWNER/REPO/pipelines?source=push",
		httpmock.NewStringResponse(http.StatusOK, `
		[
			{
				"id": 22222222,
				"iid": 4,
				"project_id": 5,
				"sha": "c9a7c0d9351cd1e71d1c2ad8277f3bc7e3c47d1f",
				"ref": "main",
				"status": "success",
				"source": "push",
				"created_at": "2020-11-30T18:20:47.571Z",
				"updated_at": "2020-11-30T18:39:40.092Z",
				"web_url": "https://gitlab.com/OWNER/REPO/-/pipelines/709793838"
			}
		]
	`))

	fakeHTTP.RegisterResponder(http.MethodDelete, "/api/v4/projects/OWNER/REPO/pipelines/22222222",
		httpmock.NewStringResponse(http.StatusNoContent, ""),
	)

	args := "--source=push"
	output, err := runCommand(fakeHTTP, args)
	require.NoError(t, err)

	out := output.String()

	assert.Equal(t, heredoc.Doc(`
		✓ Pipeline #22222222 deleted successfully.
		`), out)
	assert.Empty(t, output.Stderr())
}

func TestParseRawPipelineIDsCorrectly(t *testing.T) {
	pipelineIDs, err := parseRawPipelineIDs("1,2,3")

	require.NoError(t, err)
	assert.Equal(t, []int{1, 2, 3}, pipelineIDs)
}

func TestParseRawPipelineIDsWithError(t *testing.T) {
	pipelineIDs, err := parseRawPipelineIDs("test")

	require.Error(t, err)
	assert.Len(t, pipelineIDs, 0)
}

func TestExtractPipelineIDsFromFlagsWithError(t *testing.T) {
	fakeHTTP := httpmock.New()
	fakeHTTP.MatchURL = httpmock.PathAndQuerystring
	defer fakeHTTP.Verify(t)

	fakeHTTP.RegisterResponder(http.MethodGet, "/api/v4/projects/OWNER/REPO/pipelines?status=success",
		httpmock.NewStringResponse(http.StatusForbidden, `{message: 403 Forbidden}`))

	args := "--status=success"
	output, err := runCommand(fakeHTTP, args)
	require.Error(t, err)

	out := output.String()

	assert.Empty(t, out)
	assert.Empty(t, output.Stderr())
}

func TestOptsFromFlags(t *testing.T) {
	flags := pflag.NewFlagSet("test-flagset", pflag.ContinueOnError)
	SetupCommandFlags(flags)

	require.NoError(t, flags.Parse([]string{"--status", "success", "--older-than", "24h"}))

	opts := optsFromFlags(flags)

	assert.Nil(t, opts.Source)
	assert.Equal(t, opts.Status, gitlab.Ptr(gitlab.BuildStateValue("success")))

	lowerTimeBoundary := time.Now().Add(-1 * 24 * time.Hour).Add(-5 * time.Second)
	upperTimeBoundary := time.Now().Add(-1 * 24 * time.Hour).Add(5 * time.Second)
	assert.WithinRange(t, *opts.UpdatedBefore, lowerTimeBoundary, upperTimeBoundary)
}

func TestOptsFromFlagsWithPagination(t *testing.T) {
	flags := pflag.NewFlagSet("test-flagset", pflag.ContinueOnError)
	SetupCommandFlags(flags)

	require.NoError(t, flags.Parse([]string{"--page", "5", "--per-page", "10"}))

	opts := optsFromFlags(flags)

	assert.Equal(t, opts.Page, 5)
	assert.Equal(t, opts.PerPage, 10)
}
