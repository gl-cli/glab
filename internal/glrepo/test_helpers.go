package glrepo

import (
	"fmt"

	"github.com/xanzy/go-gitlab"
)

func TestProject(owner, repo string) Interface {
	fullname := fmt.Sprintf("%s/%s", owner, repo)
	hostname := normalizeHostname("gitlab.com")

	gitlabProject := &gitlab.Project{ID: 3}

	testRepo := glRepo{
		owner: owner, name: repo, fullname: fullname, hostname: hostname,
		project: &Project{Project: gitlabProject, fullname: fullname, hostname: hostname},
	}

	return &testRepo
}
