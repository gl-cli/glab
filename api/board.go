package api

import gitlab "gitlab.com/gitlab-org/api/client-go"

var CreateIssueBoard = func(client *gitlab.Client, projectID any, opts *gitlab.CreateIssueBoardOptions) (*gitlab.IssueBoard, error) {
	board, _, err := client.Boards.CreateIssueBoard(projectID, opts)
	if err != nil {
		return nil, err
	}

	return board, nil
}

var ListIssueBoards = func(client *gitlab.Client, projectID any, opts *gitlab.ListIssueBoardsOptions) ([]*gitlab.IssueBoard, error) {
	boards, _, err := client.Boards.ListIssueBoards(projectID, opts)
	if err != nil {
		return nil, err
	}

	return boards, nil
}

var ListGroupIssueBoards = func(client *gitlab.Client, groupID any, opts *gitlab.ListGroupIssueBoardsOptions) ([]*gitlab.GroupIssueBoard, error) {
	boards, _, err := client.GroupIssueBoards.ListGroupIssueBoards(groupID, opts)
	if err != nil {
		return nil, err
	}

	return boards, nil
}

var GetIssueBoardLists = func(client *gitlab.Client, projectID any, boardID int, opts *gitlab.GetIssueBoardListsOptions) ([]*gitlab.BoardList, error) {
	boardLists, _, err := client.Boards.GetIssueBoardLists(projectID, boardID, opts)
	if err != nil {
		return nil, err
	}

	return boardLists, nil
}

var GetGroupIssueBoardLists = func(client *gitlab.Client, groupID any, boardID int, opts *gitlab.ListGroupIssueBoardListsOptions) ([]*gitlab.BoardList, error) {
	boardLists, _, err := client.GroupIssueBoards.ListGroupIssueBoardLists(groupID, boardID, opts)
	if err != nil {
		return nil, err
	}

	return boardLists, nil
}
