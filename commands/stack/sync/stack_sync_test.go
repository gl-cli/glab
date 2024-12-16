package sync

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cli/commands/cmdtest"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/pkg/git"
	"gitlab.com/gitlab-org/cli/pkg/iostreams"
	"go.uber.org/mock/gomock"
)

type SyncScenario struct {
	refs       map[string]TestRef
	title      string
	pushNeeded bool
}

type TestRef struct {
	ref   git.StackRef
	state string
}

func setupTestFactory(rt http.RoundTripper) (*iostreams.IOStreams, *cmdutils.Factory, *Options) {
	ios, _, _, _ := cmdtest.InitIOStreams(false, "")

	f := cmdtest.InitFactory(ios, rt)

	f.BaseRepo = func() (glrepo.Interface, error) {
		return glrepo.TestProject("stack_guy", "stackproject"), nil
	}

	f.Remotes = func() (glrepo.Remotes, error) {
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

	client, _ := f.HttpClient()

	opts := &Options{
		Remotes:   f.Remotes,
		LabClient: client,
		Config:    f.Config,
		BaseRepo:  f.BaseRepo,
	}

	return ios, f, opts
}

func Test_stackSync(t *testing.T) {
	type args struct {
		stack SyncScenario
	}

	tests := []struct {
		name      string
		args      args
		httpMocks []git.HttpMock
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

			httpMocks: []git.HttpMock{
				git.MockStackUser(),
				git.MockListStackMRsByBranch("Branch1", "25"),
				git.MockGetStackMR("Branch1", "25"),
				git.MockPostStackMR("Branch2", "Branch1", "3"),
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

			httpMocks: []git.HttpMock{
				git.MockStackUser(),
				git.MockPostStackMR("Branch1", "", "3"),
				git.MockPostStackMR("Branch2", "Branch1", "3"),
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

			httpMocks: []git.HttpMock{
				git.MockStackUser(),
				git.MockListStackMRsByBranch("Branch1", "25"),
				git.MockGetStackMR("Branch1", "25"),
				git.MockPostStackMR("Branch2", "Branch1", "3"),
				git.MockPostStackMR("Branch3", "Branch2", "3"),
				git.MockPostStackMR("Branch4", "Branch3", "3"),
				git.MockPostStackMR("Branch5", "Branch4", "3"),
				git.MockPostStackMR("Branch6", "Branch5", "3"),
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			git.InitGitRepoWithCommit(t)

			fakeHTTP := git.SetupMocks(tc.httpMocks)
			defer fakeHTTP.Verify(t)

			ctrl := gomock.NewController(t)
			mockCmd := NewMockGitRunner(ctrl)

			ios, f, opts := setupTestFactory(fakeHTTP)

			err := git.SetConfig("glab.currentstack", tc.args.stack.title)
			require.NoError(t, err)

			createStack(t, tc.args.stack.title, tc.args.stack.refs)
			stack, err := git.GatherStackRefs(tc.args.stack.title)
			require.NoError(t, err)

			mockCmd.EXPECT().Git([]string{"fetch", "origin"})

			for ref := range stack.Iter() {
				state := tc.args.stack.refs[ref.SHA].state

				mockCmd.EXPECT().Git([]string{"checkout", ref.Branch}).Do(checkoutBranch(ref.Branch))
				mockCmd.EXPECT().Git([]string{"status", "-uno"}).Return(state, nil)

				switch state {
				case BranchIsBehind:
					mockCmd.EXPECT().Git([]string{"pull"}).Return(state, nil)

				case BranchHasDiverged:
					mockCmd.EXPECT().Git([]string{"checkout", stack.Last().Branch}).Do(checkoutBranch(stack.Last().Branch))
					mockCmd.EXPECT().Git([]string{"rebase", "--fork-point", "--update-refs", ref.Branch})

				case NothingToCommit:
				}

				if ref.MR == "" {
					mockCmd.EXPECT().Git([]string{"push", "--set-upstream", "origin", ref.Branch})

					if ref.IsFirst() == true {
						// this is to check for the default branch
						mockCmd.EXPECT().Git([]string{"remote", "show", "origin"}).Return("main", nil)
					}
				}
			}

			if tc.args.stack.pushNeeded {
				command := append([]string{"push", "origin", "--force-with-lease"}, stack.Branches()...)
				mockCmd.EXPECT().Git(command)
			}

			err = stackSync(f, ios, opts, mockCmd)

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

func checkoutBranch(branch string) func(_ ...string) (string, error) {
	return func(_ ...string) (string, error) {
		err := git.CheckoutBranch(branch)
		return "", err
	}
}
