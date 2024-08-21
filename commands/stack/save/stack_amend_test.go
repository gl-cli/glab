package save

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cli/commands/cmdtest"
	"gitlab.com/gitlab-org/cli/internal/run"
	"gitlab.com/gitlab-org/cli/pkg/git"
)

func Test_stackAmendCmd(t *testing.T) {
	tests := []struct {
		desc          string
		args          []string
		files         []string
		amendedFiles  []string
		description   string
		expected      string
		wantErr       bool
		editorMessage string
	}{
		{
			desc:         "amending regular files",
			args:         []string{"testfile", "randomfile"},
			files:        []string{"testfile", "randomfile"},
			amendedFiles: []string{"otherfile"},
			description:  "this is a commit message",
			expected:     "Amended stack item with description: \"this is a commit message\".\n",
		},
		{
			desc:          "with no message",
			args:          []string{"testfile", "randomfile"},
			files:         []string{"testfile", "randomfile"},
			amendedFiles:  []string{"otherfile"},
			description:   "",
			editorMessage: "amended description",
			expected:      "Amended stack item with description: \"amended description\".\n",
		},
		{
			desc:         "with no amended changes",
			args:         []string{"."},
			files:        []string{"oldfile"},
			amendedFiles: []string{},
			description:  "this is a commit message",
			expected:     "no changes to save",
			wantErr:      true,
		},
		{
			desc:         "not on a stack branch",
			args:         []string{"asdf"},
			files:        []string{"asdf"},
			amendedFiles: []string{"otherfile"},
			description:  "this is a commit message",
			expected:     "not currently in a stack",
			wantErr:      true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			ios, _, _, _ := cmdtest.InitIOStreams(true, "")
			f := cmdtest.InitFactory(ios, nil)

			dir := git.InitGitRepoWithCommit(t)
			err := git.SetLocalConfig("glab.currentstack", "cool-test-feature")
			require.Nil(t, err)

			createTemporaryFiles(t, dir, tc.files)

			var saveArgs []string
			saveArgs = append(saveArgs, "-m")
			saveArgs = append(saveArgs, "\"original save message\"")
			saveArgs = append(saveArgs, tc.args...)

			getText := getMockEditor(tc.editorMessage, &[]string{})
			_, err = runSaveCommand(nil, getText, true, strings.Join(saveArgs, " "))
			require.Nil(t, err)

			createTemporaryFiles(t, dir, tc.amendedFiles)
			if tc.desc == "not on a stack branch" {
				checkout := git.GitCommand("checkout", "-b", "randobranch")
				_, err := run.PrepareCmd(checkout).Output()

				require.Nil(t, err)
			}

			output, err := amendFunc(f, tc.args, getText, tc.description)

			if tc.wantErr {
				require.ErrorContains(t, err, tc.expected)
			} else {
				require.Nil(t, err)
				require.Equal(t, tc.expected, output)
			}
		})
	}
}
