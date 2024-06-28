package create

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/MakeNowJust/heredoc/v2"
	"gitlab.com/gitlab-org/cli/commands/cmdtest"
	"gitlab.com/gitlab-org/cli/pkg/prompt"

	"github.com/stretchr/testify/assert"
	"github.com/xanzy/go-gitlab"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/pkg/git"
	"gitlab.com/gitlab-org/cli/pkg/httpmock"
	"gitlab.com/gitlab-org/cli/test"
)

func runCommand(rt http.RoundTripper, branch string, isTTY bool, cli string) (*test.CmdOut, error) {
	ios, _, stdout, stderr := cmdtest.InitIOStreams(isTTY, "")
	pu, _ := url.Parse("https://gitlab.com/OWNER/REPO.git")

	factory := cmdtest.InitFactory(ios, rt)
	factory.Remotes = func() (glrepo.Remotes, error) {
		return glrepo.Remotes{
			{
				Remote: &git.Remote{
					Name:     "upstream",
					Resolved: "base",
					PushURL:  pu,
				},
				Repo: glrepo.New("OWNER", "REPO"),
			},
			{
				Remote: &git.Remote{
					Name:     "origin",
					Resolved: "base",
					PushURL:  pu,
				},
				Repo: glrepo.New("monalisa", "REPO"),
			},
		}, nil
	}
	factory.Branch = func() (string, error) {
		return branch, nil
	}

	// TODO: shouldn't be there but the stub doesn't work without it
	_, _ = factory.HttpClient()

	runE := func(opts *CreateOpts) error {
		opts.HeadRepo = func() (glrepo.Interface, error) {
			return glrepo.New("OWNER", "REPO"), nil
		}
		return createRun(opts)
	}

	cmd := NewCmdCreate(factory, runE)
	cmd.PersistentFlags().StringP("repo", "R", "", "")

	return cmdtest.ExecuteCommand(cmd, cli, stdout, stderr)
}

func TestNewCmdCreate_tty(t *testing.T) {
	fakeHTTP := httpmock.New()
	defer fakeHTTP.Verify(t)

	fakeHTTP.RegisterResponder(http.MethodPost, "/projects/OWNER/REPO/merge_requests",
		httpmock.NewStringResponse(http.StatusCreated, `
			{
 				"id": 1,
 				"iid": 12,
 				"project_id": 3,
 				"title": "myMRtitle",
 				"description": "myMRbody",
 				"state": "opened",
 				"target_branch": "master",
 				"source_branch": "feat-new-mr",
				"web_url": "https://gitlab.com/OWNER/REPO/-/merge_requests/12"
			}
		`),
	)
	fakeHTTP.RegisterResponder(http.MethodGet, "/projects/OWNER/REPO",
		httpmock.NewStringResponse(http.StatusOK, `
			{
 				"id": 1,
				"description": null,
				"default_branch": "master",
				"web_url": "http://gitlab.com/OWNER/REPO",
				"name": "OWNER",
				"path": "REPO",
				"merge_requests_enabled": true,
				"path_with_namespace": "OWNER/REPO"
			}
		`),
	)
	fakeHTTP.RegisterResponder(http.MethodGet, "/users",
		httpmock.NewStringResponse(http.StatusOK, `
			[{
 				"username": "testuser"
			}]
		`),
	)

	ask, teardown := prompt.InitAskStubber()
	defer teardown()

	ask.Stub([]*prompt.QuestionStub{
		{
			Name:  "confirmation",
			Value: 0,
		},
	})

	cs, csTeardown := test.InitCmdStubber()
	defer csTeardown()
	cs.Stub("HEAD branch: master\n")
	cs.Stub(heredoc.Doc(`
		deadbeef HEAD
		deadb00f refs/remotes/upstream/feat-new-mr
		deadbeef refs/remotes/origin/feat-new-mr
	`))

	cliStr := []string{
		"-t", "myMRtitle",
		"-d", "myMRbody",
		"-l", "test,bug",
		"--milestone", "1",
		"--assignee", "testuser",
	}

	cli := strings.Join(cliStr, " ")

	t.Log(cli)

	output, err := runCommand(fakeHTTP, "feat-new-mr", true, cli)
	if err != nil {
		if errors.Is(err, cmdutils.SilentError) {
			t.Errorf("Unexpected error: %q", output.Stderr())
		}
		t.Error(err)
		return
	}

	assert.Contains(t, cmdtest.FirstLine([]byte(output.String())), "!12 myMRtitle (feat-new-mr)")
	assert.Contains(t, output.Stderr(), "\nCreating merge request for feat-new-mr into master in OWNER/REPO\n\n")
	assert.Contains(t, output.String(), "https://gitlab.com/OWNER/REPO/-/merge_requests/12")
}

func TestNewCmdCreate_RelatedIssue(t *testing.T) {
	fakeHTTP := httpmock.New()
	defer fakeHTTP.Verify(t)

	fakeHTTP.RegisterResponder(http.MethodPost, "/projects/OWNER/REPO/merge_requests",
		func(req *http.Request) (*http.Response, error) {
			rb, _ := io.ReadAll(req.Body)
			assert.Contains(t, string(rb), "\"title\":\"Draft: Resolve \\\"this is a issue title\\\"")
			assert.Contains(t, string(rb), "\"description\":\"\\n\\nCloses #1\"")
			resp, _ := httpmock.NewStringResponse(http.StatusCreated, `
				{
	 				"id": 1,
	 				"iid": 12,
	 				"project_id": 3,
	 				"title": "Draft: Resolve \"this is a issue title\"",
	 				"description": "\n\nCloses #1",
	 				"state": "opened",
	 				"target_branch": "master",
	 				"source_branch": "feat-new-mr",
					"web_url": "https://gitlab.com/OWNER/REPO/-/merge_requests/12"
				}
			`)(req)
			return resp, nil
		},
	)
	fakeHTTP.RegisterResponder(http.MethodGet, "/projects/OWNER/REPO",
		httpmock.NewStringResponse(http.StatusOK, `
			{
 				"id": 1,
				"description": null,
				"default_branch": "master",
				"web_url": "http://gitlab.com/OWNER/REPO",
				"name": "OWNER",
				"path": "REPO",
				"merge_requests_enabled": true,
				"path_with_namespace": "OWNER/REPO"
			}
		`),
	)
	fakeHTTP.RegisterResponder(http.MethodGet, "/projects/OWNER/REPO/issues/1",
		httpmock.NewStringResponse(http.StatusOK, `
			{
				"id":1,
				"iid":1,
				"project_id":1,
				"title":"this is a issue title",
				"description":"issue description"
			}
		`),
	)

	cs, csTeardown := test.InitCmdStubber()
	defer csTeardown()
	cs.Stub("HEAD branch: master\n")
	cs.Stub(heredoc.Doc(`
			deadbeef HEAD
			deadb00f refs/remotes/upstream/feat-new-mr
			deadbeef refs/remotes/origin/feat-new-mr
		`))

	cliStr := []string{
		"--related-issue", "1",
		"--source-branch", "feat-new-mr",
		"--yes",
	}

	cli := strings.Join(cliStr, " ")

	t.Log(cli)

	output, err := runCommand(fakeHTTP, "feat-new-mr", true, cli)
	if err != nil {
		if errors.Is(err, cmdutils.SilentError) {
			t.Errorf("Unexpected error: %q", output.Stderr())
		}
		t.Error(err)
		return
	}
	assert.Contains(t, cmdtest.FirstLine([]byte(output.String())), `!12 Draft: Resolve "this is a issue title" (feat-new-mr)`)
	assert.Contains(t, output.Stderr(), "\nCreating draft merge request for feat-new-mr into master in OWNER/REPO\n\n")
	assert.Contains(t, output.String(), "https://gitlab.com/OWNER/REPO/-/merge_requests/12")
}

func TestNewCmdCreate_TemplateFromCommitMessages(t *testing.T) {
	fakeHTTP := httpmock.New()
	defer fakeHTTP.Verify(t)

	fakeHTTP.RegisterResponder(http.MethodPost, "/projects/OWNER/REPO/merge_requests",
		func(req *http.Request) (*http.Response, error) {
			rb, _ := io.ReadAll(req.Body)
			assert.Contains(t, string(rb), "- commit msg 1  \\n\\n")
			assert.Contains(t, string(rb), "- commit msg 2  \\ncommit body")
			resp, _ := httpmock.NewStringResponse(http.StatusCreated, `
				{
	 				"id": 1,
	 				"iid": 12,
	 				"project_id": 3,
	 				"title": "...",
	 				"description": "...",
	 				"state": "opened",
	 				"target_branch": "master",
	 				"source_branch": "feat-new-mr",
					"web_url": "https://gitlab.com/OWNER/REPO/-/merge_requests/12"
				}
			`)(req)
			return resp, nil
		},
	)
	fakeHTTP.RegisterResponder(http.MethodGet, "/projects/OWNER/REPO",
		httpmock.NewStringResponse(http.StatusOK, `
			{
				"id": 1,
				"description": null,
				"default_branch": "master",
				"web_url": "http://gitlab.com/OWNER/REPO",
				"name": "OWNER",
				"path": "REPO",
				"merge_requests_enabled": true,
				"path_with_namespace": "OWNER/REPO"
			}
		`),
	)

	ask, teardown := prompt.InitAskStubber()
	defer teardown()

	ask.Stub([]*prompt.QuestionStub{
		{
			Name:  "index",
			Value: 0,
		},
	})
	ask.Stub([]*prompt.QuestionStub{
		{
			Name:    "Description",
			Default: true,
		},
	})

	cs, csTeardown := test.InitCmdStubber()
	defer csTeardown()

	cs.Stub("HEAD branch: main\n") // git remote show <name>
	cs.Stub("/")                   // git rev-parse --show-toplevel

	// git -c log.ShowSignature=false log --pretty=format:%H,%s --cherry upstream/main...feat-new-mr
	cs.Stub(heredoc.Doc(`
			deadb00f,commit msg 2
			deadbeef,commit msg 1
		`))

	// git -c log.ShowSignature=false show -s --pretty=format:%b deadbeef
	cs.Stub("")
	// git -c log.ShowSignature=false show -s --pretty=format:%b deadb00f
	cs.Stub("commit body")

	cliStr := []string{
		"--source-branch", "feat-new-mr",
		"--title", "mr-title",
		"--yes",
	}

	cli := strings.Join(cliStr, " ")

	t.Log(cli)

	output, err := runCommand(fakeHTTP, "feat-new-mr", true, cli)
	if err != nil {
		if errors.Is(err, cmdutils.SilentError) {
			t.Errorf("Unexpected error: %q", output.Stderr())
		}
		t.Error(err)
		return
	}
}

func TestNewCmdCreate_RelatedIssueWithTitleAndDescription(t *testing.T) {
	fakeHTTP := httpmock.New()
	defer fakeHTTP.Verify(t)

	fakeHTTP.RegisterResponder(http.MethodPost, "/projects/OWNER/REPO/merge_requests",
		func(req *http.Request) (*http.Response, error) {
			rb, _ := io.ReadAll(req.Body)
			assert.Contains(t, string(rb), "\"title\":\"Draft: my custom MR title\"")
			assert.Contains(t, string(rb), "\"description\":\"my custom MR description\\n\\nCloses #1\"")
			resp, _ := httpmock.NewStringResponse(http.StatusCreated, `
				{
	 				"id": 1,
	 				"iid": 12,
	 				"project_id": 3,
	 				"title": "my custom MR title",
	 				"description": "myMRbody",
	 				"state": "opened",
	 				"target_branch": "master",
	 				"source_branch": "feat-new-mr",
					"web_url": "https://gitlab.com/OWNER/REPO/-/merge_requests/12"
				}
			`)(req)
			return resp, nil
		},
	)
	fakeHTTP.RegisterResponder(http.MethodGet, "/projects/OWNER/REPO",
		httpmock.NewStringResponse(http.StatusOK, `
			{
 				"id": 1,
				"description": null,
				"default_branch": "master",
				"web_url": "http://gitlab.com/OWNER/REPO",
				"name": "OWNER",
				"path": "REPO",
				"merge_requests_enabled": true,
				"path_with_namespace": "OWNER/REPO"
			}
		`),
	)
	fakeHTTP.RegisterResponder(http.MethodGet, "/projects/OWNER/REPO/issues/1",
		httpmock.NewStringResponse(http.StatusOK, `
			{
				"id":1,
				"iid":1,
				"project_id":1,
				"title":"this is a issue title",
				"description":"issue description"
			}
		`),
	)

	cs, csTeardown := test.InitCmdStubber()
	defer csTeardown()
	cs.Stub("HEAD branch: master\n")
	cs.Stub(heredoc.Doc(`
			deadbeef HEAD
			deadb00f refs/remotes/upstream/feat-new-mr
			deadbeef refs/remotes/origin/feat-new-mr
		`))

	cliStr := []string{
		"--title", "\"my custom MR title\"",
		"--description", "\"my custom MR description\"",
		"--related-issue", "1",
		"--source-branch", "feat-new-mr",
	}

	cli := strings.Join(cliStr, " ")

	t.Log(cli)

	output, err := runCommand(fakeHTTP, "feat-new-mr", true, cli)
	if err != nil {
		if errors.Is(err, cmdutils.SilentError) {
			t.Errorf("Unexpected error: %q", output.Stderr())
		}
		t.Error(err)
		return
	}
	assert.Contains(t, cmdtest.FirstLine([]byte(output.String())), "!12 my custom MR title (feat-new-mr)")
	assert.Contains(t, output.Stderr(), "\nCreating draft merge request for feat-new-mr into master in OWNER/REPO\n\n")
	assert.Contains(t, output.String(), "https://gitlab.com/OWNER/REPO/-/merge_requests/12")
}

func TestMRCreate_nontty_insufficient_flags(t *testing.T) {
	fakeHTTP := httpmock.New()
	defer fakeHTTP.Verify(t)

	_, err := runCommand(fakeHTTP, "test-br", false, "")
	if err == nil {
		t.Fatal("expected error")
	}

	assert.Equal(t, "--title or --fill required for non-interactive mode.", err.Error())
}

func TestMrBodyAndTitle(t *testing.T) {
	opts := &CreateOpts{
		SourceBranch:         "mr-autofill-test-br",
		TargetBranch:         "master",
		TargetTrackingBranch: "origin/master",
	}
	t.Run("", func(t *testing.T) {
		cs, csTeardown := test.InitCmdStubber()
		defer csTeardown()
		cs.Stub("d1sd2e,docs: add some changes to txt file")                           // git log
		cs.Stub("Here, I am adding some commit body.\nLittle longer\n\nResolves #1\n") // git log

		if err := mrBodyAndTitle(opts); err != nil {
			t.Errorf("unexpected error: %v", err)
			return
		}

		assert.Equal(t, "docs: add some changes to txt file", opts.Title)
		assert.Equal(t, "Here, I am adding some commit body.\nLittle longer\n\nResolves #1\n", opts.Description)
	})
	t.Run("given-title", func(t *testing.T) {
		cs, csTeardown := test.InitCmdStubber()
		defer csTeardown()

		cs.Stub("d1sd2e,docs: add some changes to txt file")
		cs.Stub("Here, I am adding some commit body.\nLittle longer\n\nResolves #1\n") // git log

		opts := *opts
		opts.Title = "docs: make some other stuff"
		if err := mrBodyAndTitle(&opts); err != nil {
			t.Errorf("unexpected error: %v", err)
			return
		}

		assert.Equal(t, "docs: make some other stuff", opts.Title)
		assert.Equal(t, `Here, I am adding some commit body.
Little longer

Resolves #1
`, opts.Description)
	})
	t.Run("given-description", func(t *testing.T) {
		cs, csTeardown := test.InitCmdStubber()
		defer csTeardown()

		cs.Stub("d1sd2e,docs: add some changes to txt file")

		opts := *opts
		opts.Description = `Make it multiple lines
like this

resolves #1
`
		if err := mrBodyAndTitle(&opts); err != nil {
			t.Errorf("unexpected error: %v", err)
			return
		}

		assert.Equal(t, "docs: add some changes to txt file", opts.Title)
		assert.Equal(t, `Make it multiple lines
like this

resolves #1
`, opts.Description)
	})
	t.Run("given-fill-commit-body", func(t *testing.T) {
		opts = &CreateOpts{
			SourceBranch:         "mr-autofill-test-br",
			TargetBranch:         "master",
			TargetTrackingBranch: "origin/master",
		}
		cs, csTeardown := test.InitCmdStubber()
		defer csTeardown()

		cs.Stub("d1sd2e,chore: some tidying\nd2asa3,docs: more changes to more things")
		cs.Stub("Here, I am adding some commit body.\nLittle longer\n\nResolves #1\n")
		cs.Stub("another body for another commit\ncloses 1234\n")

		opts := *opts
		opts.FillCommitBody = true

		if err := mrBodyAndTitle(&opts); err != nil {
			t.Errorf("unexpected error: %v", err)
			return
		}

		assert.Equal(t, "mr autofill test br", opts.Title)
		assert.Equal(t, `- docs: more changes to more things  
Here, I am adding some commit body.
Little longer  
Resolves #1

- chore: some tidying  
another body for another commit
closes 1234

`, opts.Description)
	})
}

func TestGenerateMRCompareURL(t *testing.T) {
	opts := &CreateOpts{
		Labels:        []string{"backend", "frontend"},
		Assignees:     []string{"johndoe", "janedoe"},
		Reviewers:     []string{"user", "person"},
		Milestone:     15,
		TargetProject: &gitlab.Project{ID: 100},
		SourceProject: &gitlab.Project{
			ID:     101,
			WebURL: "https://gitlab.example.com/gitlab-org/gitlab",
		},
		Title:        "Autofill tests | for this @project",
		SourceBranch: "@|calc",
		TargetBranch: "project/my-branch",
	}

	u, err := generateMRCompareURL(opts)

	expectedUrl := "https://gitlab.example.com/gitlab-org/gitlab/-/merge_requests/new?" +
		"merge_request%5Bdescription%5D=%0A%2Flabel+~backend%2C+~frontend%0A%2Fassign+johndoe%2C+janedoe%0A%2Freviewer+user%2C+person%0A%2Fmilestone+%2515&" +
		"merge_request%5Bsource_branch%5D=%40%7Ccalc&merge_request%5Bsource_project_id%5D=101&merge_request%5Btarget_branch%5D=project%2Fmy-branch&merge_request%5Btarget_project_id%5D=100&" +
		"merge_request%5Btitle%5D=Autofill+tests+%7C+for+this+%40project"

	assert.NoError(t, err)
	assert.Equal(t, expectedUrl, u)
}
