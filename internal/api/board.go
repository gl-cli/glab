package api

import gitlab "gitlab.com/gitlab-org/api/client-go"

// CreateIssueBoard returns an issue board
// Attention: this is a global variable and may be overridden in tests.
var CreateIssueBoard = func(client *gitlab.Client, projectID any, opts *gitlab.CreateIssueBoardOptions) (*gitlab.IssueBoard, error) {
	board, _, err := client.Boards.CreateIssueBoard(projectID, opts)
	if err != nil {
		return nil, err
	}

	return board, nil
}
