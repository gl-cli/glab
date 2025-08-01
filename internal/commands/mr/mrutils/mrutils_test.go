package mrutils

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/cli/internal/api"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/internal/prompt"
	"gitlab.com/gitlab-org/cli/internal/testing/cmdtest"
)

func Test_DisplayMR(t *testing.T) {
	testCases := []struct {
		name        string
		mr          *gitlab.BasicMergeRequest
		output      string
		outputToTTY bool
	}{
		{
			name: "opened",
			mr: &gitlab.BasicMergeRequest{
				IID:          1,
				State:        "opened",
				Title:        "This is open",
				SourceBranch: "main",
				WebURL:       "https://gitlab.com/gitlab-org/cli/-/merge_requests/1",
			},
			output:      "!1 This is open (main)\n https://gitlab.com/gitlab-org/cli/-/merge_requests/1\n",
			outputToTTY: true,
		},
		{
			name: "merged",
			mr: &gitlab.BasicMergeRequest{
				IID:          2,
				State:        "merged",
				Title:        "This is merged",
				SourceBranch: "main",
				WebURL:       "https://gitlab.com/gitlab-org/cli/-/merge_requests/2",
			},
			output:      "!2 This is merged (main)\n https://gitlab.com/gitlab-org/cli/-/merge_requests/2\n",
			outputToTTY: true,
		},
		{
			name: "closed",
			mr: &gitlab.BasicMergeRequest{
				IID:          3,
				State:        "closed",
				Title:        "This is closed",
				SourceBranch: "main",
				WebURL:       "https://gitlab.com/gitlab-org/cli/-/merge_requests/3",
			},
			output:      "!3 This is closed (main)\n https://gitlab.com/gitlab-org/cli/-/merge_requests/3\n",
			outputToTTY: true,
		},
		{
			name: "non-tty terse output",
			mr: &gitlab.BasicMergeRequest{
				IID:          4,
				State:        "open",
				Title:        "This shouldn't be visible",
				SourceBranch: "main",
				WebURL:       "https://gitlab.com/gitlab-org/cli/-/merge_requests/4",
			},
			output:      "https://gitlab.com/gitlab-org/cli/-/merge_requests/4",
			outputToTTY: false,
		},
	}
	streams, _, _, _ := cmdtest.TestIOStreams()
	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			got := DisplayMR(streams.Color(), tC.mr, tC.outputToTTY)
			assert.Equal(t, tC.output, got)
		})
	}
}

func Test_MRCheckErrors(t *testing.T) {
	testCases := []struct {
		name    string
		mr      *gitlab.MergeRequest
		errOpts MRCheckErrOptions
		output  string
	}{
		{
			name: "draft",
			mr: &gitlab.MergeRequest{
				BasicMergeRequest: gitlab.BasicMergeRequest{
					IID:   1,
					Draft: true,
				},
			},
			errOpts: MRCheckErrOptions{
				Draft: true,
			},
			output: "this merge request is still a draft. Run `glab mr update 1 --ready` to mark it as ready for review.",
		},
		{
			name: "pipeline",
			mr: &gitlab.MergeRequest{
				BasicMergeRequest: gitlab.BasicMergeRequest{
					IID:                       1,
					MergeWhenPipelineSucceeds: true,
				},
				Pipeline: &gitlab.PipelineInfo{
					Status: "failure",
				},
			},
			errOpts: MRCheckErrOptions{
				PipelineStatus: true,
			},
			output: "the pipeline for this merge request has failed. The pipeline must succeed before merging.",
		},
		{
			name: "merged",
			mr: &gitlab.MergeRequest{
				BasicMergeRequest: gitlab.BasicMergeRequest{
					IID:   1,
					State: "merged",
				},
			},
			errOpts: MRCheckErrOptions{
				Merged: true,
			},
			output: "this merge request has already been merged.",
		},
		{
			name: "closed",
			mr: &gitlab.MergeRequest{
				BasicMergeRequest: gitlab.BasicMergeRequest{
					IID:   1,
					State: "closed",
				},
			},
			errOpts: MRCheckErrOptions{
				Closed: true,
			},
			output: "this merge request has been closed.",
		},
		{
			name: "opened",
			mr: &gitlab.MergeRequest{
				BasicMergeRequest: gitlab.BasicMergeRequest{
					IID:   1,
					State: "opened",
				},
			},
			errOpts: MRCheckErrOptions{
				Opened: true,
			},
			output: "this merge request is already open.",
		},
		{
			name: "subscribed",
			mr: &gitlab.MergeRequest{
				BasicMergeRequest: gitlab.BasicMergeRequest{
					IID: 1,
				},
				Subscribed: true,
			},
			errOpts: MRCheckErrOptions{
				Subscribed: true,
			},
			output: "you are already subscribed to this merge request.",
		},
		{
			name: "unsubscribed",
			mr: &gitlab.MergeRequest{
				BasicMergeRequest: gitlab.BasicMergeRequest{
					IID: 1,
				},
				Subscribed: false,
			},
			errOpts: MRCheckErrOptions{
				Unsubscribed: true,
			},
			output: "you are not subscribed to this merge request.",
		},
		{
			name: "merge-privilege",
			mr: &gitlab.MergeRequest{
				BasicMergeRequest: gitlab.BasicMergeRequest{
					IID: 1,
				},
				User: struct {
					CanMerge bool "json:\"can_merge\""
				}{CanMerge: false},
			},
			errOpts: MRCheckErrOptions{
				MergePrivilege: true,
			},
			output: "you do not have permission to merge this merge request.",
		},
		{
			name: "conflicts",
			mr: &gitlab.MergeRequest{
				BasicMergeRequest: gitlab.BasicMergeRequest{
					IID:          1,
					HasConflicts: true,
				},
			},
			errOpts: MRCheckErrOptions{
				Conflict: true,
			},
			output: "merge conflicts exist. Resolve the conflicts and try again, or merge locally.",
		},
	}
	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			err := MRCheckErrors(tC.mr, tC.errOpts)
			assert.EqualError(t, err, tC.output)
		})
	}

	t.Run("nil", func(t *testing.T) {
		err := MRCheckErrors(&gitlab.MergeRequest{}, MRCheckErrOptions{})
		assert.Nil(t, err)
	})
}

func Test_getMRForBranchFails(t *testing.T) {
	baseRepo := glrepo.NewWithHost("foo", "bar", "gitlab.com")

	t.Run("API-call-failed", func(t *testing.T) {
		api.ListMRs = func(client *gitlab.Client, projectID any, opts *gitlab.ListProjectMergeRequestsOptions, listOpts ...api.CliListMROption) ([]*gitlab.BasicMergeRequest, error) {
			return nil, errors.New("API call failed")
		}

		got, err := GetMRForBranch(&gitlab.Client{}, MrOptions{baseRepo, "foo", "opened", true})
		assert.Nil(t, got)
		assert.EqualError(t, err, `failed to get open merge request for "foo": API call failed`)
	})

	t.Run("no-return", func(t *testing.T) {
		api.ListMRs = func(client *gitlab.Client, projectID any, opts *gitlab.ListProjectMergeRequestsOptions, listOpts ...api.CliListMROption) ([]*gitlab.BasicMergeRequest, error) {
			return []*gitlab.BasicMergeRequest{}, nil
		}

		got, err := GetMRForBranch(&gitlab.Client{}, MrOptions{baseRepo, "foo", "opened", true})
		assert.Nil(t, got)
		assert.EqualError(t, err, `no open merge request available for "foo"`)
	})

	t.Run("owner-no-match", func(t *testing.T) {
		api.ListMRs = func(client *gitlab.Client, projectID any, opts *gitlab.ListProjectMergeRequestsOptions, listOpts ...api.CliListMROption) ([]*gitlab.BasicMergeRequest, error) {
			return []*gitlab.BasicMergeRequest{
				{
					IID: 1,
					Author: &gitlab.BasicUser{
						Username: "profclems",
					},
				},
				{
					IID: 2,
					Author: &gitlab.BasicUser{
						Username: "maxice8",
					},
				},
			}, nil
		}

		got, err := GetMRForBranch(&gitlab.Client{}, MrOptions{baseRepo, "zemzale:foo", "opened", true})
		assert.Nil(t, got)
		assert.EqualError(t, err, `no open merge request available for "foo" owned by @zemzale`)
	})
}

func Test_getMRForBranch(t *testing.T) {
	baseRepo := glrepo.NewWithHost("foo", "bar", "gitlab.com")

	testCases := []struct {
		name   string
		input  string
		mrs    []*gitlab.BasicMergeRequest
		expect *gitlab.MergeRequest
	}{
		{
			name: "one-match",
			mrs: []*gitlab.BasicMergeRequest{
				{
					IID: 1,
					Author: &gitlab.BasicUser{
						Username: "profclems",
					},
				},
			},
			expect: &gitlab.MergeRequest{
				BasicMergeRequest: gitlab.BasicMergeRequest{
					IID: 1,
					Author: &gitlab.BasicUser{
						Username: "profclems",
					},
				},
			},
		},
		{
			name:  "owner-match",
			input: "maxice8:foo",
			mrs: []*gitlab.BasicMergeRequest{
				{
					IID: 1,
					Author: &gitlab.BasicUser{
						Username: "profclems",
					},
				},
				{
					IID: 2,
					Author: &gitlab.BasicUser{
						Username: "maxice8",
					},
				},
			},
			expect: &gitlab.MergeRequest{
				BasicMergeRequest: gitlab.BasicMergeRequest{
					IID: 2,
					Author: &gitlab.BasicUser{
						Username: "maxice8",
					},
				},
			},
		},
	}
	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			api.ListMRs = func(_ *gitlab.Client, _ any, _ *gitlab.ListProjectMergeRequestsOptions, listOpts ...api.CliListMROption) ([]*gitlab.BasicMergeRequest, error) {
				return tC.mrs, nil
			}

			got, err := GetMRForBranch(&gitlab.Client{}, MrOptions{baseRepo, tC.input, "opened", true})
			assert.NoError(t, err)

			assert.Equal(t, tC.expect.IID, got.IID)
			assert.Equal(t, tC.expect.Author.Username, got.Author.Username)
		})
	}
}

func Test_getMRForBranchPrompt(t *testing.T) {
	baseRepo := glrepo.NewWithHost("foo", "bar", "gitlab.com")

	api.ListMRs = func(client *gitlab.Client, projectID any, opts *gitlab.ListProjectMergeRequestsOptions, listOpts ...api.CliListMROption) ([]*gitlab.BasicMergeRequest, error) {
		return []*gitlab.BasicMergeRequest{
			{
				IID: 1,
				Author: &gitlab.BasicUser{
					Username: "profclems",
				},
			},
			{
				IID: 2,
				Author: &gitlab.BasicUser{
					Username: "maxice8",
				},
			},
		}, nil
	}

	t.Run("success", func(t *testing.T) {
		as, restoreAsk := prompt.InitAskStubber()
		defer restoreAsk()

		as.Stub([]*prompt.QuestionStub{
			{
				Name:  "mr",
				Value: "!1 (foo) by @profclems",
			},
		})

		got, err := GetMRForBranch(&gitlab.Client{}, MrOptions{baseRepo, "foo", "opened", true})
		assert.NoError(t, err)

		assert.Equal(t, 1, got.IID)
		assert.Equal(t, "profclems", got.Author.Username)
	})

	t.Run("error", func(t *testing.T) {
		as, restoreAsk := prompt.InitAskStubber()
		defer restoreAsk()

		as.Stub([]*prompt.QuestionStub{
			{
				Name:  "mr",
				Value: errors.New("prompt failed"),
			},
		})

		got, err := GetMRForBranch(&gitlab.Client{}, MrOptions{baseRepo, "foo", "opened", true})
		assert.Nil(t, got)
		assert.EqualError(t, err, "you must select a merge request: prompt failed")
	})
}

func Test_MRFromArgsWithOpts(t *testing.T) {
	ios, _, _, _ := cmdtest.TestIOStreams()
	ios.SetPrompt("false")

	// Create base factory for tests
	baseFactory := cmdtest.NewTestFactory(ios,
		cmdtest.WithBaseRepo("foo", "bar", ""),
		cmdtest.WithBranch("main"),
	)

	t.Run("success", func(t *testing.T) {
		t.Run("via-ID", func(t *testing.T) {
			api.GetMR = func(client *gitlab.Client, projectID any, mrID int, opts *gitlab.GetMergeRequestsOptions) (*gitlab.MergeRequest, error) {
				return &gitlab.MergeRequest{
					BasicMergeRequest: gitlab.BasicMergeRequest{
						IID:          2,
						Title:        "test mr",
						SourceBranch: "main",
					},
				}, nil
			}

			expectedRepo, err := baseFactory.BaseRepo()
			if err != nil {
				t.Skipf("failed to get base repo: %s", err)
			}

			gotMR, gotRepo, err := MRFromArgs(baseFactory, []string{"2"}, "")
			assert.NoError(t, err)

			assert.Equal(t, expectedRepo.FullName(), gotRepo.FullName())

			assert.Equal(t, 2, gotMR.IID)
			assert.Equal(t, "test mr", gotMR.Title)
			assert.Equal(t, "main", gotMR.SourceBranch)
		})
		t.Run("via-name", func(t *testing.T) {
			GetMRForBranch = func(apiClient *gitlab.Client, mrOpts MrOptions) (*gitlab.BasicMergeRequest, error) {
				return &gitlab.BasicMergeRequest{
					IID:          2,
					Title:        "test mr",
					SourceBranch: "main",
				}, nil
			}

			api.GetMR = func(client *gitlab.Client, projectID any, mrID int, opts *gitlab.GetMergeRequestsOptions) (*gitlab.MergeRequest, error) {
				return &gitlab.MergeRequest{
					BasicMergeRequest: gitlab.BasicMergeRequest{
						IID:          2,
						Title:        "test mr",
						SourceBranch: "main",
					},
				}, nil
			}

			expectedRepo, err := baseFactory.BaseRepo()
			if err != nil {
				t.Skipf("failed to get base repo: %s", err)
			}

			gotMR, gotRepo, err := MRFromArgs(baseFactory, []string{"foo"}, "")
			assert.NoError(t, err)

			assert.Equal(t, expectedRepo.FullName(), gotRepo.FullName())

			assert.Equal(t, 2, gotMR.IID)
			assert.Equal(t, "test mr", gotMR.Title)
			assert.Equal(t, "main", gotMR.SourceBranch)
		})
	})

	t.Run("fail", func(t *testing.T) {
		t.Run("HttpClient", func(t *testing.T) {
			f := cmdtest.NewTestFactory(ios,
				cmdtest.WithHttpClientError(errors.New("failed to create HttpClient")),
				cmdtest.WithBaseRepo("foo", "bar", ""),
				cmdtest.WithBranch("main"),
			)

			gotMR, gotRepo, err := MRFromArgs(f, []string{}, "")
			assert.Nil(t, gotMR)
			assert.Nil(t, gotRepo)
			assert.EqualError(t, err, "failed to create HttpClient")
		})
		t.Run("BaseRepo", func(t *testing.T) {
			f := cmdtest.NewTestFactory(ios,
				cmdtest.WithBaseRepoError(errors.New("failed to create glrepo.Interface")),
				cmdtest.WithBranch("main"),
			)

			gotMR, gotRepo, err := MRFromArgs(f, []string{}, "")
			assert.Nil(t, gotMR)
			assert.Nil(t, gotRepo)
			assert.EqualError(t, err, "failed to create glrepo.Interface")
		})
		t.Run("Branch", func(t *testing.T) {
			f := cmdtest.NewTestFactory(ios,
				cmdtest.WithBaseRepo("foo", "bar", ""),
				cmdtest.WithBranchError(errors.New("failed to get branch")),
			)

			gotMR, gotRepo, err := MRFromArgs(f, []string{}, "")
			assert.Nil(t, gotMR)
			assert.Nil(t, gotRepo)
			assert.EqualError(t, err, "failed to get branch")
		})
		t.Run("Invalid-MR-ID", func(t *testing.T) {
			gotMR, gotRepo, err := MRFromArgs(baseFactory, []string{"0"}, "")
			assert.Nil(t, gotMR)
			assert.Nil(t, gotRepo)
			assert.EqualError(t, err, "invalid merge request ID provided.")
		})
		t.Run("invalid-name", func(t *testing.T) {
			GetMRForBranch = func(_ *gitlab.Client, mrOpts MrOptions) (*gitlab.BasicMergeRequest, error) {
				return nil, fmt.Errorf("no merge requests from branch %q", mrOpts.Branch)
			}

			gotMR, gotRepo, err := MRFromArgs(baseFactory, []string{"foo"}, "")
			assert.Nil(t, gotMR)
			assert.Nil(t, gotRepo)
			assert.EqualError(t, err, `no merge requests from branch "foo"`)
		})
		t.Run("api.GetMR", func(t *testing.T) {
			api.GetMR = func(client *gitlab.Client, projectID any, mrID int, opts *gitlab.GetMergeRequestsOptions) (*gitlab.MergeRequest, error) {
				return nil, errors.New("API call failed")
			}

			gotMR, gotRepo, err := MRFromArgs(baseFactory, []string{"2"}, "")
			assert.Nil(t, gotMR)
			assert.Nil(t, gotRepo)
			assert.EqualError(t, err, "failed to get merge request 2: API call failed")
		})
	})
}

func Test_DisplayAllMRs(t *testing.T) {
	streams, _, _, _ := cmdtest.TestIOStreams()
	mrs := []*gitlab.BasicMergeRequest{
		{
			IID:          1,
			State:        "opened",
			Title:        "add tests",
			TargetBranch: "main",
			SourceBranch: "new-tests",
			References: &gitlab.IssueReferences{
				Full:     "OWNER/REPO/merge_requests/1",
				Relative: "#1",
				Short:    "#1",
			},
		},
		{
			IID:          2,
			State:        "merged",
			Title:        "fix bug",
			TargetBranch: "main",
			SourceBranch: "new-feature",
			References: &gitlab.IssueReferences{
				Full:     "OWNER/REPO/merge_requests/2",
				Relative: "#2",
				Short:    "#2",
			},
		},
		{
			IID:          1,
			State:        "closed",
			Title:        "add new feature",
			TargetBranch: "main",
			SourceBranch: "new-tests",
			References: &gitlab.IssueReferences{
				Full:     "OWNER/REPO/merge_requests/3",
				Relative: "#3",
				Short:    "#3",
			},
		},
	}

	expected := `!1	OWNER/REPO/merge_requests/1	add tests	(main) ← (new-tests)
!2	OWNER/REPO/merge_requests/2	fix bug	(main) ← (new-feature)
!1	OWNER/REPO/merge_requests/3	add new feature	(main) ← (new-tests)
`

	got := DisplayAllMRs(streams, mrs)
	assert.Equal(t, expected, got)
}

func Test_PrintMRApprovalState(t *testing.T) {
	scenarios := []struct {
		name          string
		approvalState *gitlab.MergeRequestApprovalState
		expected      string
	}{
		{
			name: "approved, min approvers met",
			approvalState: &gitlab.MergeRequestApprovalState{
				Rules: []*gitlab.MergeRequestApprovalRule{
					{
						ID:                1,
						Name:              "rule 1",
						ApprovalsRequired: 2,
						ApprovedBy: []*gitlab.BasicUser{
							{
								ID:       1,
								Username: "user1",
								Name:     "User One",
							},
							{
								ID:       2,
								Username: "user2",
								Name:     "User Two",
							},
						},
						Approved: true,
					},
				},
			},
			expected: `Rule "rule 1" sufficient approvals (2/2 required):
Name	Username	Approved
User One	user1	👍	
User Two	user2	👍	

`,
		},
		{
			name: "not-approved, min approvers not met",
			approvalState: &gitlab.MergeRequestApprovalState{
				Rules: []*gitlab.MergeRequestApprovalRule{
					{
						ID:                1,
						Name:              "rule 1",
						ApprovalsRequired: 2,
						ApprovedBy: []*gitlab.BasicUser{
							{
								ID:       1,
								Username: "user1",
								Name:     "User One",
							},
						},
						Approved: false,
					},
				},
			},
			expected: `Rule "rule 1" insufficient approvals (1/2 required):
Name	Username	Approved
User One	user1	👍	

`,
		},
		{
			name: "approved, eligible approvers",
			approvalState: &gitlab.MergeRequestApprovalState{
				Rules: []*gitlab.MergeRequestApprovalRule{
					{
						ID:                1,
						Name:              "rule 1",
						ApprovalsRequired: 2,
						EligibleApprovers: []*gitlab.BasicUser{
							{
								ID:       1,
								Username: "user1",
								Name:     "User One",
							},
							{
								ID:       2,
								Username: "user2",
								Name:     "User Two",
							},
						},
						ApprovedBy: []*gitlab.BasicUser{
							{
								ID:       1,
								Username: "user1",
								Name:     "User One",
							},
							{
								ID:       2,
								Username: "user2",
								Name:     "User Two",
							},
						},
						Approved: true,
					},
				},
			},
			expected: `Rule "rule 1" sufficient approvals (2/2 required):
Name	Username	Approved
User One	user1	👍	
User Two	user2	👍	

`,
		},
		{
			name: "not approved, missing eligible approver",
			approvalState: &gitlab.MergeRequestApprovalState{
				Rules: []*gitlab.MergeRequestApprovalRule{
					{
						ID:                1,
						Name:              "rule 1",
						ApprovalsRequired: 2,
						EligibleApprovers: []*gitlab.BasicUser{
							{
								ID:       1,
								Username: "user1",
								Name:     "User One",
							},
							{
								ID:       2,
								Username: "user2",
								Name:     "User Two",
							},
						},
						ApprovedBy: []*gitlab.BasicUser{
							{
								ID:       1,
								Username: "user1",
								Name:     "User One",
							},
						},
						Approved: false,
					},
				},
			},
			expected: `Rule "rule 1" insufficient approvals (1/2 required):
Name	Username	Approved
User One	user1	👍	
User Two	user2	-	

`,
		},
		{
			name: "deterministic output without eligible approvers",
			approvalState: &gitlab.MergeRequestApprovalState{
				Rules: []*gitlab.MergeRequestApprovalRule{
					{
						ID:                1,
						Name:              "rule 1",
						ApprovalsRequired: 2,
						ApprovedBy: []*gitlab.BasicUser{
							{
								ID:       1,
								Username: "aaa",
								Name:     "User One",
							},
							{
								ID:       2,
								Username: "zzz",
								Name:     "User Two",
							},
							{
								ID:       3,
								Username: "000",
								Name:     "User Three",
							},
							{
								ID:       4,
								Username: "xyz",
								Name:     "User Four",
							},
						},
						Approved: true,
					},
				},
			},
			expected: `Rule "rule 1" sufficient approvals (4/2 required):
Name	Username	Approved
User Three	000	👍	
User One	aaa	👍	
User Four	xyz	👍	
User Two	zzz	👍	

`,
		},
		{
			name: "deterministic output with all eligible approvers",
			approvalState: &gitlab.MergeRequestApprovalState{
				Rules: []*gitlab.MergeRequestApprovalRule{
					{
						ID:                1,
						Name:              "rule 1",
						ApprovalsRequired: 2,
						ApprovedBy: []*gitlab.BasicUser{
							{
								ID:       4,
								Username: "xyz",
								Name:     "User Four",
							},
							{
								ID:       1,
								Username: "aaa",
								Name:     "User One",
							},
							{
								ID:       2,
								Username: "zzz",
								Name:     "User Two",
							},
							{
								ID:       3,
								Username: "000",
								Name:     "User Three",
							},
						},
						EligibleApprovers: []*gitlab.BasicUser{
							{
								ID:       4,
								Username: "xyz",
								Name:     "User Four",
							},
							{
								ID:       2,
								Username: "zzz",
								Name:     "User Two",
							},
						},
						Approved: true,
					},
				},
			},
			expected: `Rule "rule 1" sufficient approvals (4/2 required):
Name	Username	Approved
User Four	xyz	👍	
User Two	zzz	👍	
User Three	000	👍	
User One	aaa	👍	

`,
		},
		{
			name: "deterministic output with no eligible approvers",
			approvalState: &gitlab.MergeRequestApprovalState{
				Rules: []*gitlab.MergeRequestApprovalRule{
					{
						ID:                1,
						Name:              "rule 1",
						ApprovalsRequired: 2,
						ApprovedBy: []*gitlab.BasicUser{
							{
								ID:       1,
								Username: "aaa",
								Name:     "User One",
							},
							{
								ID:       3,
								Username: "000",
								Name:     "User Three",
							},
						},
						EligibleApprovers: []*gitlab.BasicUser{
							{
								ID:       4,
								Username: "xyz",
								Name:     "User Four",
							},
							{
								ID:       2,
								Username: "zzz",
								Name:     "User Two",
							},
						},
						Approved: true,
					},
				},
			},
			expected: `Rule "rule 1" sufficient approvals (2/2 required):
Name	Username	Approved
User Four	xyz	-	
User Two	zzz	-	
User Three	000	👍	
User One	aaa	👍	

`,
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			streams, _, stdout, _ := cmdtest.TestIOStreams()

			PrintMRApprovalState(streams, scenario.approvalState)
			assert.Equal(t, scenario.expected, stdout.String())
		})
	}
}
