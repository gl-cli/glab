package api

import (
	"errors"
	"net/http"
	"sort"

	"github.com/xanzy/go-gitlab"
)

var ErrTodoExists = errors.New("To-do already exists.")

var ApproveMR = func(client *gitlab.Client, projectID interface{}, mrID int, opts *gitlab.ApproveMergeRequestOptions) (*gitlab.MergeRequestApprovals, error) {
	if client == nil {
		client = apiClient.Lab()
	}
	mr, _, err := client.MergeRequestApprovals.ApproveMergeRequest(projectID, mrID, opts)
	if err != nil {
		return nil, err
	}

	return mr, nil
}

var GetMRApprovalState = func(client *gitlab.Client, projectID interface{}, mrID int, opts ...gitlab.RequestOptionFunc) (*gitlab.MergeRequestApprovalState, error) {
	if client == nil {
		client = apiClient.Lab()
	}
	mrApprovals, _, err := client.MergeRequestApprovals.GetApprovalState(projectID, mrID, opts...)
	if err != nil {
		return nil, err
	}

	return mrApprovals, nil
}

var GetMR = func(client *gitlab.Client, projectID interface{}, mrID int, opts *gitlab.GetMergeRequestsOptions) (*gitlab.MergeRequest, error) {
	if client == nil {
		client = apiClient.Lab()
	}
	mr, _, err := client.MergeRequests.GetMergeRequest(projectID, mrID, opts)
	if err != nil {
		return nil, err
	}

	return mr, nil
}

// ListGroupMRs retrieves merge requests for a given group with optional filtering by assignees or reviewers.
//
// Parameters:
//   - client: A GitLab client instance.
//   - groupID: The ID or name of the group.
//   - opts: GitLab-specific options for listing group merge requests.
//   - listOpts: Optional list of arguments to filter by assignees or reviewers.
//     May be any combination of api.WithMRAssignees and api.WithMRReviewers.
//
// Returns:
//   - A slice of GitLab merge request objects and an error, if any.
//
// Example usage:
//
//	groupMRs, err := api.ListGroupMRs(client, "my-group", &gitlab.ListGroupMergeRequestsOptions{},
//		api.WithMRAssignees([]int{123}),
//		api.WithMRReviewers([]int{456, 789}))
var ListGroupMRs = func(client *gitlab.Client, projectID interface{}, opts *gitlab.ListGroupMergeRequestsOptions, listOpts ...CliListMROption) ([]*gitlab.MergeRequest, error) {
	composedListOpts := composeCliListMROptions(listOpts...)
	assigneeIds, reviewerIds := composedListOpts.assigneeIds, composedListOpts.reviewerIds

	if len(assigneeIds) > 0 || len(reviewerIds) > 0 {
		return listGroupMRsWithAssigneesOrReviewers(client, projectID, opts, assigneeIds, reviewerIds)
	} else {
		return listGroupMRsBase(client, projectID, opts)
	}
}

var listGroupMRsBase = func(client *gitlab.Client, groupID interface{}, opts *gitlab.ListGroupMergeRequestsOptions) ([]*gitlab.MergeRequest, error) {
	if client == nil {
		client = apiClient.Lab()
	}
	if opts.PerPage == 0 {
		opts.PerPage = DefaultListLimit
	}

	mrs, _, err := client.MergeRequests.ListGroupMergeRequests(groupID, opts)
	if err != nil {
		return nil, err
	}

	return mrs, nil
}

var listGroupMRsWithAssigneesOrReviewers = func(client *gitlab.Client, projectID interface{}, opts *gitlab.ListGroupMergeRequestsOptions, assigneeIds []int, reviewerIds []int) ([]*gitlab.MergeRequest, error) {
	if client == nil {
		client = apiClient.Lab()
	}
	if opts.PerPage == 0 {
		opts.PerPage = DefaultListLimit
	}

	mrMap := make(map[int]*gitlab.MergeRequest)
	for _, id := range assigneeIds {
		opts.AssigneeID = gitlab.AssigneeID(id)
		assigneeMrs, err := listGroupMRsBase(client, projectID, opts)
		if err != nil {
			return nil, err
		}
		for _, mr := range assigneeMrs {
			mrMap[mr.ID] = mr
		}
	}
	opts.AssigneeID = nil // reset because it's Assignee OR Reviewer
	for _, id := range reviewerIds {
		opts.ReviewerID = gitlab.ReviewerID(id)
		reviewerMrs, err := listGroupMRsBase(client, projectID, opts)
		if err != nil {
			return nil, err
		}
		for _, mr := range reviewerMrs {
			mrMap[mr.ID] = mr
		}
	}

	mrs := make([]*gitlab.MergeRequest, 0, len(mrMap))
	for _, mr := range mrMap {
		mrs = append(mrs, mr)
	}

	sort.Slice(mrs, func(i, j int) bool {
		return mrs[i].CreatedAt.After(*mrs[j].CreatedAt)
	})

	return mrs, nil
}

var ProjectListMROptionsToGroup = func(l *gitlab.ListProjectMergeRequestsOptions) *gitlab.ListGroupMergeRequestsOptions {
	return &gitlab.ListGroupMergeRequestsOptions{
		ListOptions:            l.ListOptions,
		State:                  l.State,
		OrderBy:                l.OrderBy,
		Sort:                   l.Sort,
		Milestone:              l.Milestone,
		View:                   l.View,
		Labels:                 l.Labels,
		NotLabels:              l.NotLabels,
		WithLabelsDetails:      l.WithLabelsDetails,
		WithMergeStatusRecheck: l.WithMergeStatusRecheck,
		CreatedAfter:           l.CreatedAfter,
		CreatedBefore:          l.CreatedBefore,
		UpdatedAfter:           l.UpdatedAfter,
		UpdatedBefore:          l.UpdatedBefore,
		Scope:                  l.Scope,
		AuthorID:               l.AuthorID,
		AssigneeID:             l.AssigneeID,
		ReviewerID:             l.ReviewerID,
		ReviewerUsername:       l.ReviewerUsername,
		MyReactionEmoji:        l.MyReactionEmoji,
		SourceBranch:           l.SourceBranch,
		TargetBranch:           l.TargetBranch,
		Search:                 l.Search,
		WIP:                    l.WIP,
	}
}

// ListMRs retrieves merge requests for a given project with optional filtering by assignees or reviewers.
//
// Parameters:
//   - client: A GitLab client instance.
//   - projectID: The ID or name of the project.
//   - opts: GitLab-specific options for listing merge requests.
//   - listOpts: Optional list of arguments to filter by assignees or reviewers.
//     May be any combination of api.WithMRAssignees and api.WithMRReviewers.
//
// Returns:
//   - A slice of GitLab merge request objects and an error, if any.
//
// Example usage:
//
//	mrs, err := api.ListMRs(client, "my-group", &gitlab.ListProjectMergeRequestsOptions{},
//		api.WithMRAssignees([]int{123, 456}),
//		api.WithMRReviewers([]int{789}))
var ListMRs = func(client *gitlab.Client, projectID interface{}, opts *gitlab.ListProjectMergeRequestsOptions, listOpts ...CliListMROption) ([]*gitlab.MergeRequest, error) {
	composedListOpts := composeCliListMROptions(listOpts...)
	assigneeIds, reviewerIds := composedListOpts.assigneeIds, composedListOpts.reviewerIds

	if len(assigneeIds) > 0 || len(reviewerIds) > 0 {
		return listMRsWithAssigneesOrReviewers(client, projectID, opts, assigneeIds, reviewerIds)
	} else {
		return listMRsBase(client, projectID, opts)
	}
}

var listMRsBase = func(client *gitlab.Client, projectID interface{}, opts *gitlab.ListProjectMergeRequestsOptions) ([]*gitlab.MergeRequest, error) {
	if client == nil {
		client = apiClient.Lab()
	}
	if opts.PerPage == 0 {
		opts.PerPage = DefaultListLimit
	}

	mrs, _, err := client.MergeRequests.ListProjectMergeRequests(projectID, opts)
	if err != nil {
		return nil, err
	}

	return mrs, nil
}

var listMRsWithAssigneesOrReviewers = func(client *gitlab.Client, projectID interface{}, opts *gitlab.ListProjectMergeRequestsOptions, assigneeIds []int, reviewerIds []int) ([]*gitlab.MergeRequest, error) {
	if client == nil {
		client = apiClient.Lab()
	}
	if opts.PerPage == 0 {
		opts.PerPage = DefaultListLimit
	}

	mrMap := make(map[int]*gitlab.MergeRequest)
	for _, id := range assigneeIds {
		opts.AssigneeID = gitlab.AssigneeID(id)
		assigneeMrs, err := listMRsBase(client, projectID, opts)
		if err != nil {
			return nil, err
		}
		for _, mr := range assigneeMrs {
			mrMap[mr.ID] = mr
		}
	}
	opts.AssigneeID = nil // reset because it's Assignee OR Reviewer
	for _, id := range reviewerIds {
		opts.ReviewerID = gitlab.ReviewerID(id)
		reviewerMrs, err := listMRsBase(client, projectID, opts)
		if err != nil {
			return nil, err
		}
		for _, mr := range reviewerMrs {
			mrMap[mr.ID] = mr
		}
	}

	mrs := make([]*gitlab.MergeRequest, 0, len(mrMap))
	for _, mr := range mrMap {
		mrs = append(mrs, mr)
	}

	sort.Slice(mrs, func(i, j int) bool {
		return mrs[i].CreatedAt.After(*mrs[j].CreatedAt)
	})

	return mrs, nil
}

var UpdateMR = func(client *gitlab.Client, projectID interface{}, mrID int, opts *gitlab.UpdateMergeRequestOptions) (*gitlab.MergeRequest, error) {
	if client == nil {
		client = apiClient.Lab()
	}
	mr, _, err := client.MergeRequests.UpdateMergeRequest(projectID, mrID, opts)
	if err != nil {
		return nil, err
	}

	return mr, nil
}

var DeleteMR = func(client *gitlab.Client, projectID interface{}, mrID int) error {
	if client == nil {
		client = apiClient.Lab()
	}
	_, err := client.MergeRequests.DeleteMergeRequest(projectID, mrID)
	if err != nil {
		return err
	}

	return nil
}

var MergeMR = func(client *gitlab.Client, projectID interface{}, mrID int, opts *gitlab.AcceptMergeRequestOptions) (*gitlab.MergeRequest, *gitlab.Response, error) {
	if client == nil {
		client = apiClient.Lab()
	}
	mrs, resp, err := client.MergeRequests.AcceptMergeRequest(projectID, mrID, opts)
	if err != nil {
		return nil, resp, err
	}

	return mrs, resp, nil
}

var CreateMR = func(client *gitlab.Client, projectID interface{}, opts *gitlab.CreateMergeRequestOptions) (*gitlab.MergeRequest, error) {
	if client == nil {
		client = apiClient.Lab()
	}
	mr, _, err := client.MergeRequests.CreateMergeRequest(projectID, opts)
	if err != nil {
		return nil, err
	}

	return mr, nil
}

var GetMRLinkedIssues = func(client *gitlab.Client, projectID interface{}, mrID int, opts *gitlab.GetIssuesClosedOnMergeOptions) ([]*gitlab.Issue, error) {
	if client == nil {
		client = apiClient.Lab()
	}
	mrIssues, _, err := client.MergeRequests.GetIssuesClosedOnMerge(projectID, mrID, opts)
	if err != nil {
		return nil, err
	}

	return mrIssues, nil
}

var CreateMRNote = func(client *gitlab.Client, projectID interface{}, mrID int, opts *gitlab.CreateMergeRequestNoteOptions) (*gitlab.Note, error) {
	if client == nil {
		client = apiClient.Lab()
	}

	note, _, err := client.Notes.CreateMergeRequestNote(projectID, mrID, opts)
	if err != nil {
		return note, err
	}

	return note, nil
}

var ListMRNotes = func(client *gitlab.Client, projectID interface{}, mrID int, opts *gitlab.ListMergeRequestNotesOptions) ([]*gitlab.Note, error) {
	if client == nil {
		client = apiClient.Lab()
	}

	if opts.PerPage == 0 {
		opts.PerPage = DefaultListLimit
	}

	notes, _, err := client.Notes.ListMergeRequestNotes(projectID, mrID, opts)
	if err != nil {
		return notes, err
	}

	return notes, nil
}

var RebaseMR = func(client *gitlab.Client, projectID interface{}, mrID int, opts *gitlab.RebaseMergeRequestOptions) error {
	if client == nil {
		client = apiClient.Lab()
	}

	_, err := client.MergeRequests.RebaseMergeRequest(projectID, mrID, opts)
	if err != nil {
		return err
	}

	return nil
}

var UnapproveMR = func(client *gitlab.Client, projectID interface{}, mrID int) error {
	if client == nil {
		client = apiClient.Lab()
	}

	_, err := client.MergeRequestApprovals.UnapproveMergeRequest(projectID, mrID)
	if err != nil {
		return err
	}

	return nil
}

var SubscribeToMR = func(client *gitlab.Client, projectID interface{}, mrID int, opts gitlab.RequestOptionFunc) (*gitlab.MergeRequest, error) {
	if client == nil {
		client = apiClient.Lab()
	}

	mr, _, err := client.MergeRequests.SubscribeToMergeRequest(projectID, mrID, opts)
	if err != nil {
		return nil, err
	}

	return mr, nil
}

var UnsubscribeFromMR = func(client *gitlab.Client, projectID interface{}, mrID int, opts gitlab.RequestOptionFunc) (*gitlab.MergeRequest, error) {
	if client == nil {
		client = apiClient.Lab()
	}

	mr, _, err := client.MergeRequests.UnsubscribeFromMergeRequest(projectID, mrID, opts)
	if err != nil {
		return nil, err
	}

	return mr, nil
}

var MRTodo = func(client *gitlab.Client, projectID interface{}, mrID int, opts gitlab.RequestOptionFunc) (*gitlab.Todo, error) {
	if client == nil {
		client = apiClient.Lab()
	}

	mr, resp, err := client.MergeRequests.CreateTodo(projectID, mrID, opts)

	if resp.StatusCode == http.StatusNotModified {
		return nil, ErrTodoExists
	}

	if err != nil {
		return nil, err
	}

	return mr, nil
}

type cliListMROptions struct {
	assigneeIds []int
	reviewerIds []int
}

type CliListMROption func(*cliListMROptions)

func WithMRAssignees(assigneeIds []int) CliListMROption {
	return func(c *cliListMROptions) {
		c.assigneeIds = assigneeIds
	}
}

func WithMRReviewers(reviewerIds []int) CliListMROption {
	return func(c *cliListMROptions) {
		c.reviewerIds = reviewerIds
	}
}

func composeCliListMROptions(optionSetters ...CliListMROption) *cliListMROptions {
	opts := &cliListMROptions{}
	for _, setter := range optionSetters {
		setter(opts)
	}
	return opts
}
