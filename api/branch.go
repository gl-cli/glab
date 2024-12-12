package api

import gitlab "gitlab.com/gitlab-org/api/client-go"

var CreateBranch = func(client *gitlab.Client, projectID interface{}, opts *gitlab.CreateBranchOptions) (*gitlab.Branch, error) {
	if client == nil {
		client = apiClient.Lab()
	}
	branch, _, err := client.Branches.CreateBranch(projectID, opts)
	if err != nil {
		return nil, err
	}

	return branch, nil
}
