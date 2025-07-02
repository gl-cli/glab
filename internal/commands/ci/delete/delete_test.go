package delete

import (
	"errors"
	"testing"
	"time"

	"github.com/spf13/pflag"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	gitlabtesting "gitlab.com/gitlab-org/api/client-go/testing"
	"go.uber.org/mock/gomock"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/cli/internal/testing/cmdtest"
)

func TestCIDelete(t *testing.T) {
	t.Parallel()

	tc := gitlabtesting.NewTestClient(t)
	tc.MockPipelines.EXPECT().DeletePipeline("OWNER/REPO", 11111111).Return(nil, nil)
	exec := cmdtest.SetupCmdForTest(t, NewCmdDelete, cmdtest.WithGitLabClient(tc.Client))

	out, err := exec("11111111")
	require.NoError(t, err)

	assert.Contains(t, out.OutBuf.String(), "Pipeline #11111111 deleted successfully.")
	assert.Empty(t, out.ErrBuf.String())
}

func TestCIDeleteNonExistingPipeline(t *testing.T) {
	t.Parallel()

	tc := gitlabtesting.NewTestClient(t)
	tc.MockPipelines.EXPECT().DeletePipeline("OWNER/REPO", 11111111).Return(nil, errors.New(`{"message": "404 Not found"}`))
	exec := cmdtest.SetupCmdForTest(t, NewCmdDelete, cmdtest.WithGitLabClient(tc.Client))

	out, err := exec("11111111")
	require.Error(t, err)
	assert.Empty(t, out.OutBuf.String())
}

func TestCIDeleteWithWrongArgument(t *testing.T) {
	t.Parallel()

	exec := cmdtest.SetupCmdForTest(t, NewCmdDelete)

	out, err := exec("test")
	require.Error(t, err)
	assert.Empty(t, out.OutBuf.String())
}

func TestCIDeleteByStatus(t *testing.T) {
	t.Parallel()

	tc := gitlabtesting.NewTestClient(t)
	gomock.InOrder(
		tc.MockPipelines.EXPECT().
			ListProjectPipelines("OWNER/REPO", &gitlab.ListProjectPipelinesOptions{Status: gitlab.Ptr(gitlab.Success)}).
			Return([]*gitlab.PipelineInfo{
				{
					ID: 11111111,
				},
				{
					ID: 22222222,
				},
			}, &gitlab.Response{NextPage: 0}, nil),
		tc.MockPipelines.EXPECT().DeletePipeline("OWNER/REPO", 11111111).Return(nil, nil),
		tc.MockPipelines.EXPECT().DeletePipeline("OWNER/REPO", 22222222).Return(nil, nil),
	)
	exec := cmdtest.SetupCmdForTest(t, NewCmdDelete, cmdtest.WithGitLabClient(tc.Client))

	out, err := exec("--status=success")
	require.NoError(t, err)

	stdout := out.OutBuf.String()
	assert.Contains(t, stdout, "Pipeline #11111111 deleted successfully.")
	assert.Contains(t, stdout, "Pipeline #22222222 deleted successfully.")
	assert.Empty(t, out.ErrBuf.String())
}

func TestCIDeleteByStatusFailsWithArgument(t *testing.T) {
	t.Parallel()

	exec := cmdtest.SetupCmdForTest(t, NewCmdDelete)
	out, err := exec("--status=success 11111111")
	assert.EqualError(t, err, "either a status filter or a pipeline ID must be passed, but not both.")

	assert.Empty(t, out.OutBuf.String())
	assert.Empty(t, out.ErrBuf.String())
}

func TestCIDeleteWithoutFilterFailsWithoutArgument(t *testing.T) {
	t.Parallel()

	exec := cmdtest.SetupCmdForTest(t, NewCmdDelete)
	out, err := exec("")
	assert.EqualError(t, err, "accepts 1 arg(s), received 0")

	assert.Empty(t, out.OutBuf.String())
	assert.Empty(t, out.ErrBuf.String())
}

func TestCIDeleteMultiple(t *testing.T) {
	t.Parallel()

	tc := gitlabtesting.NewTestClient(t)
	gomock.InOrder(
		tc.MockPipelines.EXPECT().DeletePipeline("OWNER/REPO", 11111111).Return(nil, nil),
		tc.MockPipelines.EXPECT().DeletePipeline("OWNER/REPO", 22222222).Return(nil, nil),
	)
	exec := cmdtest.SetupCmdForTest(t, NewCmdDelete, cmdtest.WithGitLabClient(tc.Client))

	out, err := exec("11111111,22222222")
	require.NoError(t, err)

	stdout := out.OutBuf.String()
	assert.Contains(t, stdout, "Pipeline #11111111 deleted successfully.")
	assert.Contains(t, stdout, "Pipeline #22222222 deleted successfully.")
	assert.Empty(t, out.ErrBuf.String())
}

func TestCIDryRunDeleteNothing(t *testing.T) {
	t.Parallel()

	exec := cmdtest.SetupCmdForTest(t, NewCmdDelete)
	out, err := exec("--dry-run 11111111,22222222")
	require.NoError(t, err)

	stdout := out.OutBuf.String()
	assert.Contains(t, stdout, "Pipeline #11111111 will be deleted.")
	assert.Contains(t, stdout, "Pipeline #22222222 will be deleted.")
	assert.Empty(t, out.ErrBuf.String())
}

func TestCIDeletedDryRunWithFilterDoesNotDelete(t *testing.T) {
	t.Parallel()

	tc := gitlabtesting.NewTestClient(t)
	tc.MockPipelines.EXPECT().
		ListProjectPipelines("OWNER/REPO", &gitlab.ListProjectPipelinesOptions{Status: gitlab.Ptr(gitlab.Success)}).
		Return([]*gitlab.PipelineInfo{
			{
				ID: 11111111,
			},
			{
				ID: 22222222,
			},
		}, &gitlab.Response{NextPage: 0}, nil)
	exec := cmdtest.SetupCmdForTest(t, NewCmdDelete, cmdtest.WithGitLabClient(tc.Client))

	out, err := exec("--dry-run --status=success")
	require.NoError(t, err)

	stdout := out.OutBuf.String()
	assert.Contains(t, stdout, "Pipeline #11111111 will be deleted.")
	assert.Contains(t, stdout, "Pipeline #22222222 will be deleted.")
	assert.Empty(t, out.ErrBuf.String())
}

func TestCIDeleteBySource(t *testing.T) {
	t.Parallel()

	tc := gitlabtesting.NewTestClient(t)
	gomock.InOrder(
		tc.MockPipelines.EXPECT().
			ListProjectPipelines("OWNER/REPO", &gitlab.ListProjectPipelinesOptions{Source: gitlab.Ptr("push")}).
			Return([]*gitlab.PipelineInfo{
				{
					ID: 22222222,
				},
			}, &gitlab.Response{NextPage: 0}, nil),
		tc.MockPipelines.EXPECT().DeletePipeline("OWNER/REPO", 22222222).Return(nil, nil),
	)
	exec := cmdtest.SetupCmdForTest(t, NewCmdDelete, cmdtest.WithGitLabClient(tc.Client))

	out, err := exec("--source=push")
	require.NoError(t, err)

	stdout := out.OutBuf.String()
	assert.Contains(t, stdout, "Pipeline #22222222 deleted successfully.")
	assert.Empty(t, out.ErrBuf.String())
}

func TestParseRawPipelineIDsCorrectly(t *testing.T) {
	t.Parallel()

	pipelineIDs, err := parseRawPipelineIDs("1,2,3")

	require.NoError(t, err)
	assert.Equal(t, []int{1, 2, 3}, pipelineIDs)
}

func TestParseRawPipelineIDsWithError(t *testing.T) {
	t.Parallel()

	pipelineIDs, err := parseRawPipelineIDs("test")

	require.Error(t, err)
	assert.Len(t, pipelineIDs, 0)
}

func TestExtractPipelineIDsFromFlagsWithError(t *testing.T) {
	t.Parallel()

	tc := gitlabtesting.NewTestClient(t)
	tc.MockPipelines.EXPECT().
		ListProjectPipelines("OWNER/REPO", &gitlab.ListProjectPipelinesOptions{Status: gitlab.Ptr(gitlab.Success)}).
		Return(nil, nil, errors.New(`{"message": "403 Forbidden"}`))
	exec := cmdtest.SetupCmdForTest(t, NewCmdDelete, cmdtest.WithGitLabClient(tc.Client))

	out, err := exec("--status=success")
	require.Error(t, err)

	assert.Empty(t, out.OutBuf.String())
	assert.Empty(t, out.ErrBuf.String())
}

func TestOptsFromFlags(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

	flags := pflag.NewFlagSet("test-flagset", pflag.ContinueOnError)
	SetupCommandFlags(flags)

	require.NoError(t, flags.Parse([]string{"--page", "5", "--per-page", "10"}))

	opts := optsFromFlags(flags)

	assert.Equal(t, opts.Page, 5)
	assert.Equal(t, opts.PerPage, 10)
}
