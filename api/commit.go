package api

import gitlab "gitlab.com/gitlab-org/api/client-go"

var GetCommitStatuses = func(client *gitlab.Client, pid interface{}, sha string) ([]*gitlab.CommitStatus, error) {
	if client == nil {
		client = apiClient.Lab()
	}

	opt := &gitlab.GetCommitStatusesOptions{
		All: gitlab.Ptr(true),
	}

	statuses, _, err := client.Commits.GetCommitStatuses(pid, sha, opt, nil)
	if err != nil {
		return nil, err
	}
	return statuses, nil
}
