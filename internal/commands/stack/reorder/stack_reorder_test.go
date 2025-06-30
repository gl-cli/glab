package reorder

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cli/internal/git"
	"gitlab.com/gitlab-org/cli/internal/run"
)

func Test_matchBranchesToStack(t *testing.T) {
	type args struct {
		stack    git.Stack
		branches []string
	}
	tests := []struct {
		name     string
		args     args
		expected git.Stack
		wantErr  bool
	}{
		{
			name: "basic situation",
			args: args{
				stack: git.Stack{
					Refs: map[string]git.StackRef{
						"123": {SHA: "123", Prev: "", Next: "456", Branch: "Branch1", Description: "blah1"},
						"456": {SHA: "456", Prev: "123", Next: "789", Branch: "Branch2", Description: "blah2"},
						"789": {SHA: "789", Prev: "456", Next: "", Branch: "Branch3", Description: "blah3"},
					},
				},
				branches: []string{"Branch2", "Branch3", "Branch1"},
			},
			expected: git.Stack{
				Refs: map[string]git.StackRef{
					"456": {SHA: "456", Prev: "", Next: "789", Branch: "Branch2", Description: "blah2"},
					"789": {SHA: "789", Prev: "456", Next: "123", Branch: "Branch3", Description: "blah3"},
					"123": {SHA: "123", Prev: "789", Next: "", Branch: "Branch1", Description: "blah1"},
				},
			},
		},

		{
			name: "missing branches from reordered list",
			args: args{
				stack: git.Stack{
					Refs: map[string]git.StackRef{
						"123": {SHA: "123", Prev: "", Next: "456", Branch: "Branch1"},
						"456": {SHA: "456", Prev: "123", Next: "789", Branch: "Branch2"},
						"789": {SHA: "789", Prev: "456", Next: "", Branch: "Branch3"},
					},
				},
				branches: []string{"Branch2", "Branch1"},
			},
			expected: git.Stack{},
			wantErr:  true,
		},

		{
			name: "large stack",
			args: args{
				stack: git.Stack{
					Refs: map[string]git.StackRef{
						"1":  {SHA: "1", Prev: "", Next: "2", Branch: "Branch1"},
						"2":  {SHA: "2", Prev: "1", Next: "3", Branch: "Branch2"},
						"3":  {SHA: "3", Prev: "2", Next: "4", Branch: "Branch3"},
						"4":  {SHA: "4", Prev: "3", Next: "5", Branch: "Branch4"},
						"5":  {SHA: "5", Prev: "4", Next: "6", Branch: "Branch5"},
						"6":  {SHA: "6", Prev: "5", Next: "7", Branch: "Branch6"},
						"7":  {SHA: "7", Prev: "6", Next: "8", Branch: "Branch7"},
						"8":  {SHA: "8", Prev: "7", Next: "9", Branch: "Branch8"},
						"9":  {SHA: "9", Prev: "8", Next: "10", Branch: "Branch9"},
						"10": {SHA: "10", Prev: "9", Next: "11", Branch: "Branch10"},
						"11": {SHA: "11", Prev: "10", Next: "12", Branch: "Branch11"},
						"12": {SHA: "12", Prev: "11", Next: "13", Branch: "Branch12"},
						"13": {SHA: "13", Prev: "12", Next: "", Branch: "Branch13"},
					},
				},
				branches: []string{
					"Branch12",
					"Branch1",
					"Branch2",
					"Branch8",
					"Branch11",
					"Branch3",
					"Branch6",
					"Branch9",
					"Branch7",
					"Branch5",
					"Branch10",
					"Branch13",
					"Branch4",
				},
			},
			expected: git.Stack{
				Refs: map[string]git.StackRef{
					"12": {SHA: "12", Prev: "", Next: "1", Branch: "Branch12"},
					"1":  {SHA: "1", Prev: "12", Next: "2", Branch: "Branch1"},
					"2":  {SHA: "2", Prev: "1", Next: "8", Branch: "Branch2"},
					"8":  {SHA: "8", Prev: "2", Next: "11", Branch: "Branch8"},
					"11": {SHA: "11", Prev: "8", Next: "3", Branch: "Branch11"},
					"3":  {SHA: "3", Prev: "11", Next: "6", Branch: "Branch3"},
					"6":  {SHA: "6", Prev: "3", Next: "9", Branch: "Branch6"},
					"9":  {SHA: "9", Prev: "6", Next: "7", Branch: "Branch9"},
					"7":  {SHA: "7", Prev: "9", Next: "5", Branch: "Branch7"},
					"5":  {SHA: "5", Prev: "7", Next: "10", Branch: "Branch5"},
					"10": {SHA: "10", Prev: "5", Next: "13", Branch: "Branch10"},
					"13": {SHA: "13", Prev: "10", Next: "4", Branch: "Branch13"},
					"4":  {SHA: "4", Prev: "13", Next: "", Branch: "Branch4"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			git.InitGitRepo(t)

			err := git.CreateRefFiles(tt.args.stack.Refs, "cool stack")
			require.Nil(t, err)

			git.CreateBranches(t, tt.args.branches)

			newStack, err := matchBranchesToStack(tt.args.stack, tt.args.branches)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				for k, ref := range tt.expected.Refs {
					require.Equal(t, newStack.Refs[k], ref)
				}

				require.Equal(t, len(tt.args.branches), len(newStack.Refs))
			}
		})
	}
}

func Test_updateMRs(t *testing.T) {
	type args struct {
		newStack  git.Stack
		oldStack  git.Stack
		httpMocks []git.HttpMock
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "update a complex stack",
			args: args{
				newStack: git.Stack{
					Refs: map[string]git.StackRef{
						"7": {
							SHA: "7", Prev: "", Next: "5", Branch: "Branch7",
							MR: "http://gitlab.com/stack_guy/stackproject/-/merge_requests/7",
						},
						"5": {
							SHA: "5", Prev: "7", Next: "8", Branch: "Branch5",
							MR: "http://gitlab.com/stack_guy/stackproject/-/merge_requests/5",
						},
						"8": {
							SHA: "8", Prev: "5", Next: "1", Branch: "Branch8",
							MR: "http://gitlab.com/stack_guy/stackproject/-/merge_requests/8",
						},
						"1": {
							SHA: "1", Prev: "8", Next: "9", Branch: "Branch1",
							MR: "http://gitlab.com/stack_guy/stackproject/-/merge_requests/1",
						},
						"9": {
							SHA: "9", Prev: "1", Next: "4", Branch: "Branch9",
							MR: "http://gitlab.com/stack_guy/stackproject/-/merge_requests/9",
						},
						"4": {
							SHA: "4", Prev: "9", Next: "2", Branch: "Branch4",
							MR: "http://gitlab.com/stack_guy/stackproject/-/merge_requests/4",
						},
						"2": {
							SHA: "2", Prev: "4", Next: "3", Branch: "Branch2",
							MR: "http://gitlab.com/stack_guy/stackproject/-/merge_requests/2",
						},
						"3": {
							SHA: "3", Prev: "2", Next: "6", Branch: "Branch3",
							MR: "http://gitlab.com/stack_guy/stackproject/-/merge_requests/3",
						},
						"6": {
							SHA: "6", Prev: "3", Next: "10", Branch: "Branch6",
							MR: "http://gitlab.com/stack_guy/stackproject/-/merge_requests/6",
						},
						"10": {
							SHA: "10", Prev: "6", Next: "12", Branch: "Branch10",
							MR: "http://gitlab.com/stack_guy/stackproject/-/merge_requests/10",
						},
						"12": {
							SHA: "12", Prev: "10", Next: "11", Branch: "Branch12",
							MR: "",
						},
						"11": {
							SHA: "11", Prev: "12", Next: "13", Branch: "Branch11",
							MR: "",
						},
						"13": {
							SHA: "13", Prev: "11", Next: "", Branch: "Branch13",
							MR: "",
						},
					},
				},

				oldStack: git.Stack{
					Refs: map[string]git.StackRef{
						"1": {
							SHA: "1", Prev: "", Next: "2", Branch: "Branch1",
							MR: "http://gitlab.com/stack_guy/stackproject/-/merge_requests/1",
						},
						"2": {
							SHA: "2", Prev: "1", Next: "3", Branch: "Branch2",
							MR: "http://gitlab.com/stack_guy/stackproject/-/merge_requests/2",
						},
						"3": {
							SHA: "3", Prev: "2", Next: "4", Branch: "Branch3",
							MR: "http://gitlab.com/stack_guy/stackproject/-/merge_requests/3",
						},
						"4": {
							SHA: "4", Prev: "3", Next: "5", Branch: "Branch4",
							MR: "http://gitlab.com/stack_guy/stackproject/-/merge_requests/4",
						},
						"5": {
							SHA: "5", Prev: "4", Next: "6", Branch: "Branch5",
							MR: "http://gitlab.com/stack_guy/stackproject/-/merge_requests/5",
						},
						"6": {
							SHA: "6", Prev: "5", Next: "7", Branch: "Branch6",
							MR: "http://gitlab.com/stack_guy/stackproject/-/merge_requests/6",
						},
						"7": {
							SHA: "7", Prev: "6", Next: "8", Branch: "Branch7",
							MR: "http://gitlab.com/stack_guy/stackproject/-/merge_requests/7",
						},
						"8": {
							SHA: "8", Prev: "7", Next: "9", Branch: "Branch8",
							MR: "http://gitlab.com/stack_guy/stackproject/-/merge_requests/8",
						},
						"9": {
							SHA: "9", Prev: "8", Next: "10", Branch: "Branch9",
							MR: "http://gitlab.com/stack_guy/stackproject/-/merge_requests/9",
						},
						"10": {
							SHA: "10", Prev: "9", Next: "11", Branch: "Branch10",
							MR: "http://gitlab.com/stack_guy/stackproject/-/merge_requests/10",
						},
						"11": {
							SHA: "11", Prev: "10", Next: "12", Branch: "Branch11",
							MR: "",
						},
						"12": {
							SHA: "12", Prev: "11", Next: "13", Branch: "Branch12",
							MR: "",
						},
						"13": {
							SHA: "13", Prev: "12", Next: "", Branch: "Branch13",
							MR: "",
						},
					},
				},
				httpMocks: []git.HttpMock{
					git.MockListOpenStackMRsByBranch("Branch7", "7"),
					git.MockGetStackMR("Branch7", "7"),
					git.MockPutStackMR("main", "7", "3"),

					git.MockListOpenStackMRsByBranch("Branch5", "5"),
					git.MockGetStackMR("Branch5", "5"),
					git.MockPutStackMR("Branch7", "5", "3"),

					git.MockListOpenStackMRsByBranch("Branch8", "8"),
					git.MockGetStackMR("Branch8", "8"),
					git.MockPutStackMR("Branch5", "8", "3"),

					git.MockListOpenStackMRsByBranch("Branch1", "1"),
					git.MockGetStackMR("Branch1", "1"),
					git.MockPutStackMR("Branch8", "1", "3"),

					git.MockListOpenStackMRsByBranch("Branch9", "9"),
					git.MockGetStackMR("Branch9", "9"),
					git.MockPutStackMR("Branch1", "9", "3"),

					git.MockListOpenStackMRsByBranch("Branch4", "4"),
					git.MockGetStackMR("Branch4", "4"),
					git.MockPutStackMR("Branch9", "4", "3"),

					git.MockListOpenStackMRsByBranch("Branch2", "2"),
					git.MockGetStackMR("Branch2", "2"),
					git.MockPutStackMR("Branch4", "2", "3"),

					git.MockListOpenStackMRsByBranch("Branch3", "3"),
					git.MockGetStackMR("Branch3", "3"),
					git.MockPutStackMR("Branch2", "3", "3"),

					git.MockListOpenStackMRsByBranch("Branch6", "6"),
					git.MockGetStackMR("Branch6", "6"),
					git.MockPutStackMR("Branch3", "6", "3"),

					git.MockListOpenStackMRsByBranch("Branch10", "10"),
					git.MockGetStackMR("Branch10", "10"),
					git.MockPutStackMR("Branch6", "10", "3"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			git.InitGitRepoWithCommit(t)

			gitAddRemote := git.GitCommand("remote", "add", "origin", "http://gitlab.com/gitlab-org/cli.git")
			_, err := run.PrepareCmd(gitAddRemote).Output()
			require.NoError(t, err)

			fakeHTTP := git.SetupMocks(tt.args.httpMocks)
			defer fakeHTTP.Verify(t)

			_, factory := setupTestFactory(fakeHTTP, false)

			err = updateMRs(factory, tt.args.newStack, tt.args.oldStack)

			require.NoError(t, err)
		})
	}
}
