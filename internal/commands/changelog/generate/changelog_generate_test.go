package generate

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	gitlab "gitlab.com/gitlab-org/api/client-go"
	gitlabtesting "gitlab.com/gitlab-org/api/client-go/testing"
	"gitlab.com/gitlab-org/cli/internal/testing/cmdtest"
)

func TestChangelogGenerate(t *testing.T) {
	tc := gitlabtesting.NewTestClient(t)
	gomock.InOrder(
		tc.MockProjects.EXPECT().
			GetProject("OWNER/REPO", gomock.Any()).
			Return(&gitlab.Project{ID: 37777023}, nil, nil),
		tc.MockRepositories.EXPECT().
			GenerateChangelogData(37777023, gitlab.GenerateChangelogDataOptions{Version: gitlab.Ptr("1.0.0")}).
			Return(&gitlab.ChangelogData{
				Notes: "## 1.0.0 (2023-04-02)\n\n### FirstName LastName firstname@lastname.com (1 changes)\n\n- [inital commit](gitlab-org/cli@somehash ([merge request](gitlab-org/cli!1))\n",
			}, nil, nil),
	)

	exec := cmdtest.SetupCmdForTest(
		t,
		NewCmdGenerate,
		cmdtest.WithGitLabClient(tc.Client),
	)

	out, err := exec("--version 1.0.0")
	require.Nil(t, err)

	assert.Empty(t, out.ErrBuf.String())

	expectedStr := "## 1.0.0 (2023-04-02)\n\n### FirstName LastName firstname@lastname.com (1 changes)\n\n- [inital commit](gitlab-org/cli@somehash ([merge request](gitlab-org/cli!1))\n"
	assert.Equal(t, expectedStr, out.OutBuf.String())
}

func TestChangelogGenerateWithError(t *testing.T) {
	cases := map[string]struct {
		httpStatus  int
		httpMsgJSON string
		errorMsg    string
	}{
		"unauthorized": {
			httpStatus:  http.StatusUnauthorized,
			httpMsgJSON: "{message: 401 Unauthorized}",
			errorMsg:    "GET https://gitlab.com/api/v4/projects/37777023/repository/changelog: 401 failed to parse unknown error format: {message: 401 Unauthorized}",
		},
		"not found": {
			httpStatus:  http.StatusNotFound,
			httpMsgJSON: "{message: 404 Project Not Found}",
			errorMsg:    "404 Not Found",
		},
	}

	for name, v := range cases {
		t.Run(name, func(t *testing.T) {
			tc := gitlabtesting.NewTestClient(t)
			gomock.InOrder(
				tc.MockProjects.EXPECT().
					GetProject("OWNER/REPO", gomock.Any()).
					Return(&gitlab.Project{ID: 37777023}, nil, nil),
				tc.MockRepositories.EXPECT().
					GenerateChangelogData(37777023, gitlab.GenerateChangelogDataOptions{Version: gitlab.Ptr("1.0.0")}).
					Return(nil, nil, errors.New(v.errorMsg)),
			)

			exec := cmdtest.SetupCmdForTest(
				t,
				NewCmdGenerate,
				cmdtest.WithGitLabClient(tc.Client),
			)

			_, err := exec("--version 1.0.0")
			require.NotNil(t, err)
		})
	}
}
