package update

import (
	"fmt"
	"testing"
	"time"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/google/shlex"

	"gitlab.com/gitlab-org/cli/internal/config"

	"github.com/acarl005/stripansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/cli/internal/api"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/testing/cmdtest"
)

// TODO: test by mocking the appropriate api function
func TestMain(m *testing.M) {
	cmdtest.InitTest(m, "mr_update_test")
}

func TestUpdateMergeRequest(t *testing.T) {
	io, _, stdout, stderr := cmdtest.TestIOStreams(cmdtest.WithTestIOStreamsAsTTY(true))
	f := cmdtest.NewTestFactory(io, cmdtest.WithConfig(config.NewFromString(heredoc.Doc(`
		hosts:
		  gitlab.com:
		    username: monalisa
		    token: OTOKEN
	`))))
	oldUpdateMr := api.UpdateMR
	timer, _ := time.Parse(time.RFC3339, "2014-11-12T11:45:26.371Z")
	toggle := false

	api.UpdateMR = func(client *gitlab.Client, projectID any, mrID int, opts *gitlab.UpdateMergeRequestOptions) (*gitlab.MergeRequest, error) {
		if projectID == "" || projectID == "WRONG_REPO" || projectID == "expected_err" || mrID == 0 {
			return nil, fmt.Errorf("error expected")
		}
		repo, err := f.BaseRepo()
		if err != nil {
			return nil, err
		}

		mr := &gitlab.MergeRequest{
			BasicMergeRequest: gitlab.BasicMergeRequest{
				ID:          mrID,
				IID:         mrID,
				Title:       *opts.Title,
				Labels:      gitlab.Labels{"bug", "test"},
				State:       "opened",
				Description: "mrbody",
				Author: &gitlab.BasicUser{
					ID:       mrID,
					Name:     "John Dev Wick",
					Username: "jdwick",
				},
				WebURL:    "https://" + repo.RepoHost() + "/" + repo.FullName() + "/-/merge_requests/" + fmt.Sprintf("%d", mrID),
				CreatedAt: &timer,
			},
		}

		if opts.RemoveSourceBranch != nil {
			toggle = !toggle
			mr.ForceRemoveSourceBranch = toggle
		}

		return mr, nil
	}

	api.GetMR = func(client *gitlab.Client, projectID any, mrID int, opts *gitlab.GetMergeRequestsOptions) (*gitlab.MergeRequest, error) {
		if projectID == "" || projectID == "WRONG_REPO" || projectID == "expected_err" {
			return nil, fmt.Errorf("error expected")
		}
		repo, err := f.BaseRepo()
		if err != nil {
			return nil, err
		}
		title := map[int]string{
			1: "mrTitle",
			2: "Draft: mrTitle",
			3: "Draft: wip: wip: draft: DrAfT: mrTitle",
		}[mrID]
		return &gitlab.MergeRequest{
			BasicMergeRequest: gitlab.BasicMergeRequest{
				ID:                      mrID,
				IID:                     mrID,
				Title:                   title,
				Labels:                  gitlab.Labels{"test", "bug"},
				State:                   "opened",
				Description:             "mrBody",
				ForceRemoveSourceBranch: toggle,
				Author: &gitlab.BasicUser{
					ID:       mrID,
					Name:     "John Dev Wick",
					Username: "jdwick",
				},
				WebURL: fmt.Sprintf("https://%s/%s/-/merge_requests/%d", repo.RepoHost(), repo.FullName(), mrID),
			},
		}, nil
	}

	api.ListMRs = func(client *gitlab.Client, projectID any, opts *gitlab.ListProjectMergeRequestsOptions, listOpts ...api.CliListMROption) ([]*gitlab.BasicMergeRequest, error) {
		return []*gitlab.BasicMergeRequest{}, nil
	}

	testCases := []struct {
		Name        string
		Args        string
		ExpectedMsg []string
		wantErr     bool
	}{
		{
			Name:        "Update works",
			Args:        "1",
			ExpectedMsg: []string{"- Updating merge request !1"},
		},
		{
			Name: "Remove source branch",
			Args: "1 --remove-source-branch",
			ExpectedMsg: []string{
				"- Updating merge request !1",
				"✓ enabled removal of source branch on merge",
			},
		},
		{
			Name: "Restore remove source branch",
			Args: "1 --remove-source-branch",
			ExpectedMsg: []string{
				"- Updating merge request !1",
				"✓ disabled removal of source branch on merge",
			},
		},
		{
			Name:        "Issue Does Not Exist",
			Args:        "0",
			ExpectedMsg: []string{"- Updating merge request !0", "error expected"},
			wantErr:     true,
		},
		{
			Name:        "Set draft",
			Args:        "1 --draft",
			ExpectedMsg: []string{"marked as Draft", "!1 Draft: mrTitle"},
		},
		{
			Name:        "Don't set draft twice",
			Args:        "2 --draft",
			ExpectedMsg: []string{"✓ already a Draft", "2 Draft: mrTitle"},
		},
		{
			Name:        "Set ready",
			Args:        "2 --ready",
			ExpectedMsg: []string{"✓ marked as ready", "2 mrTitle"},
		},
		{
			Name:        "Set ready with multiple draft prefixes",
			Args:        "3 --ready",
			ExpectedMsg: []string{"✓ marked as ready", "3 mrTitle"},
		},
	}

	cumulativePreviousOutputLen := 0
	for _, tc := range testCases {
		cmd := NewCmdUpdate(f)
		cmdutils.EnableRepoOverride(cmd, f)
		t.Run(tc.Name, func(t *testing.T) {
			argv, err := shlex.Split(tc.Args)
			if err != nil {
				t.Fatal(err)
			}
			cmd.SetArgs(argv)
			_, err = cmd.ExecuteC()
			if err != nil {
				if tc.wantErr {
					require.Error(t, err)
					return
				} else {
					t.Fatal(err)
				}
			}

			out := stripansi.Strip(stdout.String())[cumulativePreviousOutputLen:]
			cumulativePreviousOutputLen += len(out)

			for _, msg := range tc.ExpectedMsg {
				assert.Contains(t, out, msg)
				assert.Equal(t, "", stderr.String())
			}
		})
	}

	api.UpdateMR = oldUpdateMr
}
