package compile

import (
	"errors"
	"path"
	"testing"

	gitlab "gitlab.com/gitlab-org/api/client-go"
	gitlabtesting "gitlab.com/gitlab-org/api/client-go/testing"
	"gitlab.com/gitlab-org/cli/internal/testing/cmdtest"
	"go.uber.org/mock/gomock"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_compileRun(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                 string
		testFile             string
		StdOut               string
		wantErr              bool
		errMsg               string
		expectedLintResponse *gitlab.ProjectLintResult
		showHaveBaseRepo     bool
	}{
		{
			name:             "with invalid path specified",
			testFile:         "WRONG_PATH",
			StdOut:           "",
			wantErr:          true,
			errMsg:           "WRONG_PATH: no such file or directory",
			showHaveBaseRepo: true,
		},
		{
			name:             "without base repo",
			testFile:         ".gitlab.ci.yml",
			StdOut:           "",
			wantErr:          true,
			errMsg:           "You must be in a GitLab project repository for this action: no base repository present",
			showHaveBaseRepo: false,
		},
		{
			name:             "when a valid path is specified and yaml is valid",
			testFile:         ".gitlab-ci.yml",
			StdOut:           "",
			wantErr:          false,
			errMsg:           "",
			showHaveBaseRepo: true,
			expectedLintResponse: &gitlab.ProjectLintResult{
				Valid: true,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tc := gitlabtesting.NewTestClient(t)
			gomock.InOrder(
				tc.MockProjects.EXPECT().GetProject("OWNER/REPO", gomock.Any()).Return(&gitlab.Project{ID: 123}, nil, nil).MaxTimes(1),
				tc.MockValidate.EXPECT().ProjectNamespaceLint(123, gomock.Any()).Return(tt.expectedLintResponse, nil, nil).MaxTimes(1),
			)
			options := []cmdtest.FactoryOption{cmdtest.WithGitLabClient(tc.Client)}
			if !tt.showHaveBaseRepo {
				options = append(options, cmdtest.WithBaseRepoError(errors.New("no base repository present")))
			}
			exec := cmdtest.SetupCmdForTest(t, NewCmdConfigCompile, options...)

			args := path.Join(cmdtest.ProjectPath, "test/testdata", tt.testFile)
			out, err := exec(args)
			if tt.wantErr {
				require.Contains(t, err.Error(), tt.errMsg)
				return
			}
			require.NoError(t, err)

			assert.Equal(t, tt.StdOut, out.OutBuf.String())
		})
	}
}
