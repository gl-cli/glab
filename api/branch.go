package api

import gitlab "gitlab.com/gitlab-org/api/client-go"

var CreateBranch = func(client *gitlab.Client, projectID any, opts *gitlab.CreateBranchOptions) (*gitlab.Branch, error) {
	branch, _, err := client.Branches.CreateBranch(projectID, opts)
	if err != nil {
		return nil, err
	}

	return branch, nil
}
