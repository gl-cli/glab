package reorder

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/git"
	"gitlab.com/gitlab-org/cli/internal/testing/cmdtest"
)

func Test_promptForReorder(t *testing.T) {
	type args struct {
		stack git.Stack
		input string
		noTTY bool
	}
	tests := []struct {
		name    string
		args    args
		want    []string
		wantErr bool
	}{
		{
			name: "Prompt correctly parses from rebase file",
			want: []string{"hello", "hello2", "hello3"},
			args: args{
				input: "hello\nhello2\nhello3",
				stack: git.Stack{},
			},
		},
		{
			name: "Getting a prompt with noTTY returns an error",
			args: args{
				input: "hello",
				noTTY: true,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isTTY := !tt.args.noTTY
			factory := setupTestFactory(t, nil, isTTY)

			prompts := []string{}
			getText := getMockEditor(tt.args.input, &prompts)

			got, err := promptForOrder(factory, getText, tt.args.stack, "")
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.Nil(t, err)
				require.Equal(t, tt.want, got)
			}
		})
	}
}

func Test_hasComment(t *testing.T) {
	type args struct {
		words []string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "has comment",
			args: args{words: []string{"branch", "#", "hello", "hey"}},
			want: true,
		},
		{
			name: "has no comment",
			args: args{words: []string{"branch", "hello", "hey"}},
			want: false,
		},
		{
			name: "with empty arguments",
			args: args{words: []string{}},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, hasComment(tt.args.words))
		})
	}
}

func Test_parseReorderFile(t *testing.T) {
	type args struct {
		input string
	}
	tests := []struct {
		name         string
		args         args
		wantBranches []string
		wantErr      bool
	}{
		{
			name: "A regular file parses correctly",
			args: args{
				input: "hello\nhello2\nhello3",
			},
			wantBranches: []string{"hello", "hello2", "hello3"},
		},
		{
			name: "A file with comments parses correctly",
			args: args{
				input: "hello\n#sneakycomment!\nhello2\nhello3",
			},
			wantBranches: []string{"hello", "hello2", "hello3"},
		},
		{
			name: "A file with unexpected text after the branch gives an error",
			args: args{
				input: "hello i'm a very long branch\nhello2\nhello3",
			},
			wantBranches: []string{},
			wantErr:      true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotBranches, err := parseReorderFile(tt.args.input)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, gotBranches, tt.wantBranches)
		})
	}
}

func setupTestFactory(t *testing.T, rt http.RoundTripper, isTTY bool) cmdutils.Factory {
	ios, _, _, _ := cmdtest.TestIOStreams(cmdtest.WithTestIOStreamsAsTTY(isTTY))

	f := cmdtest.NewTestFactory(ios,
		cmdtest.WithGitLabClient(cmdtest.NewTestApiClient(t, &http.Client{Transport: rt}, "", "gitlab.com").Lab()),
		cmdtest.WithBaseRepo("stack_guy", "stackproject"),
	)

	return f
}

func getMockEditor(input string, prompts *[]string) cmdutils.GetTextUsingEditor {
	return func(editor, tmpFileName, content string) (string, error) {
		*prompts = append(*prompts, content)
		return input, nil
	}
}
