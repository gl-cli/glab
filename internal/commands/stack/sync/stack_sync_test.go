package sync

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/git"
	git_testing "gitlab.com/gitlab-org/cli/internal/git/testing"
	"gitlab.com/gitlab-org/cli/internal/glinstance"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/internal/testing/cmdtest"
	"gitlab.com/gitlab-org/cli/internal/testing/gitmock"
	"go.uber.org/mock/gomock"
)

type SyncScenario struct {
	refs       map[string]TestRef
	title      string
	baseBranch string
	pushNeeded bool
}

type TestRef struct {
	ref   git.StackRef
	state string
}

func setupTestFactory(t *testing.T, rt http.RoundTripper) (cmdutils.Factory, *options) {
	ios, _, _, _ := cmdtest.TestIOStreams()

	f := cmdtest.NewTestFactory(ios,
		cmdtest.WithGitLabClient(cmdtest.NewTestApiClient(t, &http.Client{Transport: rt}, "", glinstance.DefaultHostname).Lab()),
		func(f *cmdtest.Factory) {
			f.BaseRepoStub = func() (glrepo.Interface, error) {
				return glrepo.TestProject("stack_guy", "stackproject"), nil
			}
		},
		func(f *cmdtest.Factory) {
			f.RemotesStub = func() (glrepo.Remotes, error) {
				r := glrepo.Remotes{
					&glrepo.Remote{
						Remote: &git.Remote{
							Name:     "origin",
							Resolved: "head: gitlab.com/stack_guy/stackproject",
						},
						Repo: glrepo.TestProject("stack_guy", "stackproject"),
					},
				}
				return r, nil
			}
		},
	)

	client, _ := f.HttpClient()

	return f, &options{
		io:        ios,
		remotes:   f.Remotes,
		labClient: client,
		baseRepo:  f.BaseRepo,
	}
}

func Test_stackSync(t *testing.T) {
	type args struct {
		stack SyncScenario
	}

	tests := []struct {
		name      string
		args      args
		httpMocks []gitmock.HttpMock
		wantErr   bool
	}{
		{
			name: "two branches, 1st branch has MR, 2nd branch behind",
			args: args{
				stack: SyncScenario{
					title: "my cool stack",
					refs: map[string]TestRef{
						"1": {
							ref: git.StackRef{
								SHA: "1", Prev: "", Next: "2", Branch: "Branch1",
								MR: "http://gitlab.com/stack_guy/stackproject/-/merge_requests/1",
							},
							state: NothingToCommit,
						},
						"2": {
							ref:   git.StackRef{SHA: "2", Prev: "1", Next: "", Branch: "Branch2", MR: ""},
							state: BranchIsBehind,
						},
					},
				},
			},

			httpMocks: []gitmock.HttpMock{
				gitmock.MockStackUser(),
				gitmock.MockListStackMRsByBranch("Branch1", "25"),
				gitmock.MockGetStackMR("Branch1", "25"),
				gitmock.MockPostStackMR("Branch2", "Branch1", "3"),
			},
		},

		{
			name: "two branches, no MRs, nothing to commit",
			args: args{
				stack: SyncScenario{
					title: "my cool stack",
					refs: map[string]TestRef{
						"1": {
							ref:   git.StackRef{SHA: "1", Prev: "", Next: "2", Branch: "Branch1", MR: ""},
							state: NothingToCommit,
						},
						"2": {
							ref:   git.StackRef{SHA: "2", Prev: "1", Next: "", Branch: "Branch2", MR: ""},
							state: NothingToCommit,
						},
					},
				},
			},

			httpMocks: []gitmock.HttpMock{
				gitmock.MockStackUser(),
				gitmock.MockPostStackMR("Branch1", "main", "3"),
				gitmock.MockPostStackMR("Branch2", "Branch1", "3"),
			},
		},

		{
			name: "a complicated scenario",
			args: args{
				stack: SyncScenario{
					title:      "my cool stack",
					pushNeeded: true,
					refs: map[string]TestRef{
						"1": {
							ref: git.StackRef{
								SHA: "1", Prev: "", Next: "2", Branch: "Branch1",
								MR: "http://gitlab.com/stack_guy/stackproject/-/merge_requests/1",
							},
							state: NothingToCommit,
						},
						"2": {
							ref:   git.StackRef{SHA: "2", Prev: "1", Next: "3", Branch: "Branch2", MR: ""},
							state: NothingToCommit,
						},
						"3": {
							ref:   git.StackRef{SHA: "3", Prev: "2", Next: "4", Branch: "Branch3", MR: ""},
							state: NothingToCommit,
						},
						"4": {
							ref:   git.StackRef{SHA: "4", Prev: "3", Next: "5", Branch: "Branch4", MR: ""},
							state: BranchHasDiverged,
						},
						"5": {
							ref:   git.StackRef{SHA: "5", Prev: "4", Next: "6", Branch: "Branch5", MR: ""},
							state: NothingToCommit,
						},
						"6": {
							ref:   git.StackRef{SHA: "6", Prev: "5", Next: "", Branch: "Branch6", MR: ""},
							state: NothingToCommit,
						},
					},
				},
			},

			httpMocks: []gitmock.HttpMock{
				gitmock.MockStackUser(),
				gitmock.MockListStackMRsByBranch("Branch1", "25"),
				gitmock.MockGetStackMR("Branch1", "25"),
				gitmock.MockPostStackMR("Branch2", "Branch1", "3"),
				gitmock.MockPostStackMR("Branch3", "Branch2", "3"),
				gitmock.MockPostStackMR("Branch4", "Branch3", "3"),
				gitmock.MockPostStackMR("Branch5", "Branch4", "3"),
				gitmock.MockPostStackMR("Branch6", "Branch5", "3"),
			},
		},
		{
			name: "non standard base branch",
			args: args{
				stack: SyncScenario{
					title:      "my cool stack",
					baseBranch: "jawn",
					refs: map[string]TestRef{
						"1": {
							ref:   git.StackRef{SHA: "1", Prev: "", Next: "2", Branch: "Branch1", MR: ""},
							state: BranchIsBehind,
						},
						"2": {
							ref:   git.StackRef{SHA: "2", Prev: "1", Next: "", Branch: "Branch2", MR: ""},
							state: BranchIsBehind,
						},
					},
				},
			},

			httpMocks: []gitmock.HttpMock{
				gitmock.MockStackUser(),
				gitmock.MockPostStackMR("Branch1", "jawn", "3"),
				gitmock.MockPostStackMR("Branch2", "Branch1", "3"),
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			git.InitGitRepoWithCommit(t)

			fakeHTTP := gitmock.SetupMocks(tc.httpMocks)
			defer fakeHTTP.Verify(t)

			ctrl := gomock.NewController(t)
			mockCmd := git_testing.NewMockGitRunner(ctrl)

			f, opts := setupTestFactory(t, fakeHTTP)

			err := git.SetConfig("glab.currentstack", tc.args.stack.title)
			require.NoError(t, err)

			createStack(t, tc.args.stack.title, tc.args.stack.refs)
			stack, err := git.GatherStackRefs(tc.args.stack.title)
			require.NoError(t, err)

			mockCmd.EXPECT().Git([]string{"fetch", "origin"})

			for ref := range stack.Iter() {
				state := tc.args.stack.refs[ref.SHA].state

				mockCmd.EXPECT().Git([]string{"checkout", ref.Branch})
				mockCmd.EXPECT().Git([]string{"status", "-uno"}).Return(state, nil)

				switch state {
				case BranchIsBehind:
					mockCmd.EXPECT().Git([]string{"pull"}).Return(state, nil)

				case BranchHasDiverged:
					mockCmd.EXPECT().Git([]string{"checkout", stack.Last().Branch})
					mockCmd.EXPECT().Git([]string{"rebase", "--fork-point", "--update-refs", ref.Branch})

				case NothingToCommit:
				}

				if ref.MR == "" {
					if ref.IsFirst() == true {
						if tc.args.stack.baseBranch != "" {
							err := git.AddStackBaseBranch(tc.args.stack.title, tc.args.stack.baseBranch)
							require.NoError(t, err)
							mockCmd.EXPECT().Git([]string{"ls-remote", "--exit-code", "--heads", "origin", tc.args.stack.baseBranch})
						} else {
							// this is to check for the default branch
							mockCmd.EXPECT().Git([]string{"remote", "show", "origin"}).Return("HEAD branch: main", nil)
							mockCmd.EXPECT().Git([]string{"ls-remote", "--exit-code", "--heads", "origin", "main"})
						}
					}

					mockCmd.EXPECT().Git([]string{"push", "--set-upstream", "origin", ref.Branch}).Return("a", nil)

				}
			}

			if tc.args.stack.pushNeeded {
				command := append([]string{"push", "origin", "--force-with-lease"}, stack.Branches()...)
				mockCmd.EXPECT().Git(command)
			}

			err = opts.run(f, mockCmd)

			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func createStack(t *testing.T, title string, scenario map[string]TestRef) {
	t.Helper()
	_ = git.CheckoutNewBranch("main")

	for _, ref := range scenario {
		err := git.AddStackRefFile(title, ref.ref)
		require.NoError(t, err)

		err = git.CheckoutNewBranch(ref.ref.Branch)
		require.NoError(t, err)
	}
}
