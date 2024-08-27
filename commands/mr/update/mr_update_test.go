package update

import (
	"fmt"
	"testing"
	"time"

	"gitlab.com/gitlab-org/cli/pkg/iostreams"

	"github.com/google/shlex"

	"gitlab.com/gitlab-org/cli/internal/config"

	"github.com/acarl005/stripansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xanzy/go-gitlab"
	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/commands/cmdtest"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
)

// TODO: test by mocking the appropriate api function
func TestMain(m *testing.M) {
	cmdtest.InitTest(m, "mr_update_test")
}

func TestUpdateMergeRequest(t *testing.T) {
	defer config.StubConfig(`---
hosts:
  gitlab.com:
    username: monalisa
    token: OTOKEN
`, "")()
	io, _, stdout, stderr := iostreams.Test()
	stubFactory, _ := cmdtest.StubFactoryWithConfig("")
	stubFactory.IO = io
	stubFactory.IO.IsaTTY = true
	stubFactory.IO.IsErrTTY = true
	oldUpdateMr := api.UpdateMR
	timer, _ := time.Parse(time.RFC3339, "2014-11-12T11:45:26.371Z")
	toggle := false

	api.UpdateMR = func(client *gitlab.Client, projectID interface{}, mrID int, opts *gitlab.UpdateMergeRequestOptions) (*gitlab.MergeRequest, error) {
		if projectID == "" || projectID == "WRONG_REPO" || projectID == "expected_err" || mrID == 0 {
			return nil, fmt.Errorf("error expected")
		}
		repo, err := stubFactory.BaseRepo()
		if err != nil {
			return nil, err
		}

		mr := &gitlab.MergeRequest{
			ID:          1,
			IID:         1,
			Title:       "mrtitile",
			Labels:      gitlab.Labels{"bug", "test"},
			State:       "opened",
			Description: "mrbody",
			Author: &gitlab.BasicUser{
				ID:       1,
				Name:     "John Dev Wick",
				Username: "jdwick",
			},
			WebURL:    "https://" + repo.RepoHost() + "/" + repo.FullName() + "/-/merge_requests/1",
			CreatedAt: &timer,
		}

		if opts.RemoveSourceBranch != nil {
			toggle = !toggle
			mr.ForceRemoveSourceBranch = toggle
		}

		return mr, nil
	}

	api.GetMR = func(client *gitlab.Client, projectID interface{}, mrID int, opts *gitlab.GetMergeRequestsOptions) (*gitlab.MergeRequest, error) {
		if projectID == "" || projectID == "WRONG_REPO" || projectID == "expected_err" {
			return nil, fmt.Errorf("error expected")
		}
		repo, err := stubFactory.BaseRepo()
		if err != nil {
			return nil, err
		}
		return &gitlab.MergeRequest{
			ID:                      mrID,
			IID:                     mrID,
			Title:                   "mrTitle",
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
		}, nil
	}

	api.ListMRs = func(client *gitlab.Client, projectID interface{}, opts *gitlab.ListProjectMergeRequestsOptions, listOpts ...api.CliListMROption) ([]*gitlab.MergeRequest, error) {
		return []*gitlab.MergeRequest{}, nil
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
	}

	cmd := NewCmdUpdate(stubFactory)
	cmdutils.EnableRepoOverride(cmd, stubFactory)

	for _, tc := range testCases {
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

			out := stripansi.Strip(stdout.String())

			for _, msg := range tc.ExpectedMsg {
				assert.Contains(t, out, msg)
				assert.Equal(t, "", stderr.String())
			}
		})
	}

	api.UpdateMR = oldUpdateMr
}
