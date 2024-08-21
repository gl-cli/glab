package save

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cli/commands/cmdtest"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/internal/run"
	"gitlab.com/gitlab-org/cli/pkg/git"
	"gitlab.com/gitlab-org/cli/pkg/iostreams"
	"gitlab.com/gitlab-org/cli/test"
)

func setupTestFactory(rt http.RoundTripper, isTTY bool) (ios *iostreams.IOStreams, stdout *bytes.Buffer, stderr *bytes.Buffer, factory *cmdutils.Factory) {
	ios, _, stdout, stderr = cmdtest.InitIOStreams(isTTY, "")

	factory = cmdtest.InitFactory(ios, rt)

	_, _ = factory.HttpClient()

	return
}

func runSaveCommand(rt http.RoundTripper, getText cmdutils.GetTextUsingEditor, isTTY bool, args string) (*test.CmdOut, error) {
	_, stdout, stderr, factory := setupTestFactory(rt, isTTY)
	cmd := NewCmdSaveStack(factory, getText)

	return cmdtest.ExecuteCommand(cmd, args, stdout, stderr)
}

func TestSaveNewStack(t *testing.T) {
	tests := []struct {
		desc          string
		args          []string
		files         []string
		message       string
		expected      string
		wantErr       bool
		noTTY         bool
		editorMessage string
	}{
		{
			desc:     "adding regular files",
			args:     []string{"testfile", "randomfile"},
			files:    []string{"testfile", "randomfile"},
			message:  "this is a commit message",
			expected: "• cool-test-feature: Saved with message: \"this is a commit message\".\n",
		},

		{
			desc:     "adding files with a dot argument",
			args:     []string{"."},
			files:    []string{"testfile", "randomfile"},
			message:  "this is a commit message",
			expected: "• cool-test-feature: Saved with message: \"this is a commit message\".\n",
		},

		{
			desc:          "omitting a message",
			args:          []string{"."},
			files:         []string{"testfile"},
			editorMessage: "oh ok fine how about blah blah",
			expected:      "• cool-test-feature: Saved with message: \"oh ok fine how about blah blah\".\n",
		},

		{
			desc:     "with no changed files",
			args:     []string{"."},
			files:    []string{},
			expected: "could not save: \"no changes to save.\"",
			wantErr:  true,
		},

		{
			desc:     "Test with no message and noTTY",
			args:     []string{"."},
			files:    []string{"testfile"},
			expected: "glab stack save without `-m` and without a TTY should throw an error.",
			wantErr:  true,
			noTTY:    true,
		},
	}

	for _, tc := range tests {
		isTTY := !tc.noTTY
		t.Run(tc.desc, func(t *testing.T) {
			if tc.message != "" && isTTY {
				tc.args = append(tc.args, "-m")
				tc.args = append(tc.args, "\""+tc.message+"\"")
			}

			dir := git.InitGitRepoWithCommit(t)
			err := git.SetLocalConfig("glab.currentstack", "cool-test-feature")
			require.Nil(t, err)

			createTemporaryFiles(t, dir, tc.files)

			getText := getMockEditor(tc.editorMessage, &[]string{})
			args := strings.Join(tc.args, " ")

			output, err := runSaveCommand(nil, getText, isTTY, args)

			if tc.wantErr {
				require.Errorf(t, err, tc.expected)
			} else {
				require.Nil(t, err)
				require.Equal(t, tc.expected, output.String())
			}
		})
	}
}

func Test_addFiles(t *testing.T) {
	tests := []struct {
		desc     string
		args     []string
		expected []string
	}{
		{
			desc:     "adding regular files",
			args:     []string{"file1", "file2"},
			expected: []string{"file1", "file2"},
		},
		{
			desc:     "adding files with a dot argument",
			args:     []string{"."},
			expected: []string{"file1", "file2"},
		},
		{
			desc:     "adding files with no argument",
			expected: []string{"file1", "file2"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			dir := git.InitGitRepoWithCommit(t)
			err := git.SetLocalConfig("glab.currentstack", "cool-test-feature")
			require.Nil(t, err)

			createTemporaryFiles(t, dir, tc.expected)

			_, err = addFiles(tc.args)
			require.Nil(t, err)

			gitCmd := git.GitCommand("status", "--short", "-u")
			output, err := run.PrepareCmd(gitCmd).Output()
			require.Nil(t, err)

			normalizedFiles := []string{}
			for _, file := range tc.expected {
				file = "A  " + file

				normalizedFiles = append(normalizedFiles, file)
			}

			formattedOutput := strings.Replace(string(output), "\n", "", -1)
			require.Equal(t, formattedOutput, strings.Join(normalizedFiles, ""))
		})
	}
}

func Test_checkForChanges(t *testing.T) {
	tests := []struct {
		desc     string
		args     []string
		expected bool
	}{
		{
			desc:     "check for changes with modified files",
			args:     []string{"file1", "file2"},
			expected: true,
		},
		{
			desc:     "check for changes without anything",
			args:     []string{},
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			dir := git.InitGitRepoWithCommit(t)
			err := git.SetLocalConfig("glab.currentstack", "cool-test-feature")
			require.Nil(t, err)

			createTemporaryFiles(t, dir, tc.args)

			err = checkForChanges()
			if tc.expected {
				require.Nil(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func Test_commitFiles(t *testing.T) {
	tests := []struct {
		name    string
		want    string
		message string
		wantErr bool
	}{
		{
			name:    "a regular commit message",
			message: "i am a test message",
			want:    "i am a test message\n 2 files changed, 0 insertions(+), 0 deletions(-)\n create mode 100644 test\n create mode 100644 yo\n",
		},
		{
			name:    "no message",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := git.InitGitRepoWithCommit(t)

			createTemporaryFiles(t, dir, []string{"yo", "test"})
			_, err := addFiles([]string{"."})
			require.Nil(t, err)

			got, err := commitFiles(tt.message)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.Nil(t, err)
				require.Contains(t, got, tt.want)
			}
		})
	}
}

func Test_generateStackSha(t *testing.T) {
	type args struct {
		message   string
		title     string
		author    string
		timestamp time.Time
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "basic test",
			args: args{message: "hello", title: "supercool stack title", author: "norm maclean", timestamp: time.Date(1998, time.July, 6, 1, 3, 3, 7, time.UTC)},
			want: "e062296a",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			git.InitGitRepo(t)

			got, err := generateStackSha(tt.args.message, tt.args.title, tt.args.author, tt.args.timestamp)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.Nil(t, err)
				require.Equal(t, got, tt.want)
			}
		})
	}
}

func Test_createShaBranch(t *testing.T) {
	type args struct {
		sha   string
		title string
	}
	tests := []struct {
		name     string
		args     args
		prefix   string
		want     string
		wantErr  bool
		noConfig bool
	}{
		{
			name:   "standard test case",
			args:   args{sha: "237ec83c", title: "cool-change"},
			prefix: "asdf",
			want:   "asdf-cool-change-237ec83c",
		},
		{
			name:     "with no config file",
			args:     args{sha: "237ec83c", title: "cool-change"},
			prefix:   "",
			want:     "jawn-cool-change-237ec83c",
			noConfig: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			git.InitGitRepo(t)

			defer config.StubWriteConfig(io.Discard, io.Discard)()

			factory := createFactoryWithConfig("branch_prefix", tt.prefix)

			if tt.noConfig {
				t.Setenv("USER", "jawn")
			}

			got, err := createShaBranch(factory, tt.args.sha, tt.args.title)
			require.Nil(t, err)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.Nil(t, err)
				require.Equal(t, tt.want, got)
			}
		})
	}
}

func createTemporaryFiles(t *testing.T, dir string, files []string) {
	for _, file := range files {
		file = path.Join(dir, file)
		_, err := os.Create(file)

		require.Nil(t, err)
	}
}

func createFactoryWithConfig(key string, value string) *cmdutils.Factory {
	strconfig := heredoc.Doc(`
				` + key + `: ` + value + `
			`)

	cfg := config.NewFromString(strconfig)

	ios, _, _, _ := iostreams.Test()

	return &cmdutils.Factory{
		IO: ios,
		Config: func() (config.Config, error) {
			return cfg, nil
		},
	}
}
