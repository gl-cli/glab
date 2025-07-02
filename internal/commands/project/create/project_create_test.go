package create

import (
	"testing"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/acarl005/stripansi"
	"github.com/stretchr/testify/assert"
	gitlab "gitlab.com/gitlab-org/api/client-go"

	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/internal/testing/cmdtest"
)

func TestMain(m *testing.M) {
	cmdtest.InitTest(m, "project_create_test")
}

func Test_projectCreateCmd(t *testing.T) {
	t.Parallel()

	io, _, stdout, stderr := cmdtest.TestIOStreams()
	f := cmdtest.NewTestFactory(io, cmdtest.WithConfig(config.NewFromString(heredoc.Doc(`
		hosts:
		  gitlab.com:
		    username: monalisa
		    token: OTOKEN
		no_prompt: true
	`))))
	// to skip creation of local project directory, set prompt to false

	createProject = func(client *gitlab.Client, opts *gitlab.CreateProjectOptions) (*gitlab.Project, error) {
		return &gitlab.Project{
			ID:                1,
			Name:              *opts.Name,
			NameWithNamespace: *opts.Name,
		}, nil
	}

	currentUser = func(client *gitlab.Client) (*gitlab.User, error) {
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

	cmd := NewCmdCreate(f)
	cmdutils.EnableRepoOverride(cmd, f)

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
