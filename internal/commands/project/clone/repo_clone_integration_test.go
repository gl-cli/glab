package clone

import (
	"fmt"
	"strings"
	"testing"

	"gitlab.com/gitlab-org/cli/internal/api"
	"gitlab.com/gitlab-org/cli/internal/testing/cmdtest"

	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/test"

	"github.com/stretchr/testify/assert"
)

func Test_repoClone_Integration(t *testing.T) {
	glTestHost := test.GetHostOrSkip(t)
	t.Setenv("GITLAB_HOST", glTestHost)

	io, stdin, stdout, stderr := cmdtest.TestIOStreams()
	fac := cmdtest.NewTestFactory(io,
		func(f *cmdtest.Factory) {
			f.ApiClientStub = func(repoHost string, cfg config.Config) (*api.Client, error) {
				return api.NewClientFromConfig(repoHost, cfg, false, "glab test client")
			}
		},
	)

	cs, restore := test.InitCmdStubber()
	// git clone
	cs.Stub("")
	// git remote add
	cs.Stub("")
	defer restore()

	cmd := NewCmdClone(fac, nil)
	out, err := runCommand(cmd, "gitlab-org/cli", stdin, stdout, stderr)
	if err != nil {
		t.Errorf("unexpected error: %q", err)
		return
	}

	assert.Equal(t, "", out.String())
	assert.Equal(t, "", out.Stderr())
	assert.Equal(t, 1, cs.Count)
	assert.Regexp(t, "git clone git@gitlab.com:.*/cli.git", strings.Join(cs.Calls[0].Args, " "))
}

func Test_repoClone_group_Integration(t *testing.T) {
	names := []string{"cli-automated-testing/test", "cli-automated-testing/homebrew-testing"}
	urls := []string{"git@gitlab.com:cli-automated-testing/test.git", "git@gitlab.com:cli-automated-testing/homebrew-testing.git"}
	repoCloneTest(t, names, urls, 0, false)
}

func Test_repoClone_group_single_Integration(t *testing.T) {
	names := []string{"cli-automated-testing/test"}
	urls := []string{"git@gitlab.com:cli-automated-testing/test.git"}
	repoCloneTest(t, names, urls, 1, false)
}

func Test_repoClone_group_paginate_Integration(t *testing.T) {
	names := []string{"cli-automated-testing/test", "cli-automated-testing/homebrew-testing"}
	urls := []string{"git@gitlab.com:cli-automated-testing/test.git", "git@gitlab.com:cli-automated-testing/homebrew-testing.git"}
	repoCloneTest(t, names, urls, 1, true)
}

func repoCloneTest(t *testing.T, expectedRepoNames []string, expectedRepoUrls []string, perPage int, paginate bool) {
	assert.Equal(t, len(expectedRepoNames), len(expectedRepoUrls))

	glTestHost := test.GetHostOrSkip(t)
	t.Setenv("GITLAB_HOST", glTestHost)

	io, stdin, stdout, stderr := cmdtest.TestIOStreams()
	fac := cmdtest.NewTestFactory(io,
		func(f *cmdtest.Factory) {
			f.ApiClientStub = func(repoHost string, cfg config.Config) (*api.Client, error) {
				return api.NewClientFromConfig(repoHost, cfg, false, "glab test client")
			}
		},
	)

	cs, restore := test.InitCmdStubber()
	for range expectedRepoUrls {
		cs.Stub("")
	}

	defer restore()

	cmd := NewCmdClone(fac, nil)
	cli := "-g cli-automated-testing"
	if perPage != 0 {
		cli += fmt.Sprintf(" --per-page %d", perPage)
	}
	if paginate {
		cli += " --paginate"
	}

	// TODO: stub api.ListGroupProjects endpoint
	out, err := runCommand(cmd, cli, stdin, stdout, stderr)
	if err != nil {
		t.Errorf("unexpected error: %q", err)
		return
	}

	assert.Equal(t, "✓ "+strings.Join(expectedRepoNames, "\n✓ ")+"\n", out.String())
	assert.Equal(t, "", out.Stderr())
	assert.Equal(t, len(expectedRepoUrls), cs.Count)

	for i := range expectedRepoUrls {
		assert.Equal(t, fmt.Sprintf("git clone %s", expectedRepoUrls[i]), strings.Join(cs.Calls[i].Args, " "))
	}
}
