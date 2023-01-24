package create

import (
	"testing"

	"github.com/acarl005/stripansi"
	"github.com/stretchr/testify/assert"
	"github.com/xanzy/go-gitlab"

	"gitlab.com/gitlab-org/cli/api"

	"gitlab.com/gitlab-org/cli/commands/cmdtest"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/pkg/iostreams"
)

func TestMain(m *testing.M) {
	cmdtest.InitTest(m, "project_create_test")
}

func Test_projectCreateCmd(t *testing.T) {
	t.Parallel()
	defer config.StubConfig(`---
hosts:
  gitlab.com:
    username: monalisa
    token: OTOKEN
no_prompt: true
`, "")()

	io, _, stdout, stderr := iostreams.Test()
	stubFactory, _ := cmdtest.StubFactoryWithConfig("")
	// to skip creation of local project directory, set prompt to false
	stubFactory.IO = io
	stubFactory.IO.IsaTTY = false
	stubFactory.IO.IsErrTTY = false

	api.CreateProject = func(client *gitlab.Client, opts *gitlab.CreateProjectOptions) (*gitlab.Project, error) {
		return &gitlab.Project{
			ID:                1,
			Name:              *opts.Name,
			NameWithNamespace: *opts.Name,
		}, nil
	}

	api.CurrentUser = func(client *gitlab.Client) (*gitlab.User, error) {
		return &gitlab.User{
			ID:       1,
			Username: "username",
			Name:     "name",
		}, nil
	}

	testCases := []struct {
		Name        string
		Args        []string
		ExpectedMsg []string
		wantErr     bool
	}{
		{
			Name:        "Create project with only repo name",
			Args:        []string{"reponame"},
			ExpectedMsg: []string{"✓ Created repository reponame on GitLab: \n"},
		},
		{
			Name:        "Create project with only repo name and slash suffix",
			Args:        []string{"reponame/"},
			ExpectedMsg: []string{"✓ Created repository reponame on GitLab: \n"},
		},
	}

	cmd := NewCmdCreate(stubFactory)
	cmdutils.EnableRepoOverride(cmd, stubFactory)

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			cmd.SetArgs(tc.Args)
			_, err := cmd.ExecuteC()
			if err != nil {
				t.Fatal(err)
			}

			out := stripansi.Strip(stdout.String())

			for _, msg := range tc.ExpectedMsg {
				assert.Contains(t, out, msg)
				assert.Equal(t, "", stderr.String())
			}
		})
	}
}
