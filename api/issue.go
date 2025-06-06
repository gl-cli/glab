// This is a silly wrapper for gitlab client-go but helps maintain consistency
package api

import (
	"net/http"

	gitlab "gitlab.com/gitlab-org/api/client-go"
)

var ListIssueNotes = func(client *gitlab.Client, projectID any, issueID int, opts *gitlab.ListIssueNotesOptions) ([]*gitlab.Note, error) {
	if opts.PerPage == 0 {
		opts.PerPage = DefaultListLimit
	}
	notes, _, err := client.Notes.ListIssueNotes(projectID, issueID, opts)
	if err != nil {
		return nil, err
	}
	return notes, nil
}

var UpdateIssue = func(client *gitlab.Client, projectID any, issueID int, opts *gitlab.UpdateIssueOptions) (*gitlab.Issue, error) {
	issue, _, err := client.Issues.UpdateIssue(projectID, issueID, opts)
	if err != nil {
		return nil, err
	}

	return issue, nil
}

var GetIssue = func(client *gitlab.Client, projectID any, issueID int) (*gitlab.Issue, error) {
	issue, _, err := client.Issues.GetIssue(projectID, issueID)
	if err != nil {
		return nil, err
	}

	return issue, nil
}

var ProjectListIssueOptionsToGroup = func(l *gitlab.ListProjectIssuesOptions) *gitlab.ListGroupIssuesOptions {
	var assigneeID *gitlab.AssigneeIDValue
	if l.AssigneeID != nil {
		assigneeID = gitlab.AssigneeID(*l.AssigneeID)
	}
	return &gitlab.ListGroupIssuesOptions{
		ListOptions:        l.ListOptions,
		State:              l.State,
		Labels:             l.Labels,
		NotLabels:          l.NotLabels,
		WithLabelDetails:   l.WithLabelDetails,
		IIDs:               l.IIDs,
		Milestone:          l.Milestone,
		Scope:              l.Scope,
		AuthorID:           l.AuthorID,
		NotAuthorID:        l.NotAuthorID,
		AssigneeID:         assigneeID,
		NotAssigneeID:      l.NotAssigneeID,
		AssigneeUsername:   l.AssigneeUsername,
		MyReactionEmoji:    l.MyReactionEmoji,
		NotMyReactionEmoji: l.NotMyReactionEmoji,
		OrderBy:            l.OrderBy,
		Sort:               l.Sort,
		Search:             l.Search,
		In:                 l.In,
		CreatedAfter:       l.CreatedAfter,
		CreatedBefore:      l.CreatedBefore,
		UpdatedAfter:       l.UpdatedAfter,
		UpdatedBefore:      l.UpdatedBefore,
		IssueType:          l.IssueType,
	}
}

var ListGroupIssues = func(client *gitlab.Client, groupID any, opts *gitlab.ListGroupIssuesOptions) ([]*gitlab.Issue, error) {
	if opts.PerPage == 0 {
		opts.PerPage = DefaultListLimit
	}
	issues, _, err := client.Issues.ListGroupIssues(groupID, opts)
	if err != nil {
		return nil, err
	}

	return issues, nil
}

var ListProjectIssues = func(client *gitlab.Client, projectID any, opts *gitlab.ListProjectIssuesOptions) ([]*gitlab.Issue, error) {
	if opts.PerPage == 0 {
		opts.PerPage = DefaultListLimit
	}
	issues, _, err := client.Issues.ListProjectIssues(projectID, opts)
	if err != nil {
		return nil, err
	}

	return issues, nil
}

var CreateIssue = func(client *gitlab.Client, projectID any, opts *gitlab.CreateIssueOptions) (*gitlab.Issue, error) {
	issue, _, err := client.Issues.CreateIssue(projectID, opts)
	if err != nil {
		return nil, err
	}

	return issue, nil
}

var DeleteIssue = func(client *gitlab.Client, projectID any, issueID int) error {
	_, err := client.Issues.DeleteIssue(projectID, issueID)
	if err != nil {
		return err
	}

	return nil
}

var CreateIssueNote = func(client *gitlab.Client, projectID any, mrID int, opts *gitlab.CreateIssueNoteOptions) (*gitlab.Note, error) {
	note, _, err := client.Notes.CreateIssueNote(projectID, mrID, opts)
	if err != nil {
		return note, err
	}

	return note, nil
}

var SubscribeToIssue = func(client *gitlab.Client, projectID any, issueID int, opts gitlab.RequestOptionFunc) (*gitlab.Issue, error) {
	issue, resp, err := client.Issues.SubscribeToIssue(projectID, issueID, opts)
	if err != nil {
		if resp != nil {
			// If the user is already subscribed to the issue, the status code 304 is returned.
			if resp.StatusCode == http.StatusNotModified {
				return nil, ErrIssuableUserAlreadySubscribed
			}
		}
		return issue, err
	}

	return issue, nil
}

var UnsubscribeFromIssue = func(client *gitlab.Client, projectID any, issueID int, opts gitlab.RequestOptionFunc) (*gitlab.Issue, error) {
	issue, resp, err := client.Issues.UnsubscribeFromIssue(projectID, issueID, opts)
	if err != nil {
		if resp != nil {
			// If the user is not subscribed to the issue, the status code 304 is returned.
			if resp.StatusCode == http.StatusNotModified {
				return nil, ErrIssuableUserNotSubscribed
			}
		}
		return issue, err
	}

	return issue, nil
}

var LinkIssues = func(client *gitlab.Client, projectID any, issueIDD int, opts *gitlab.CreateIssueLinkOptions) (*gitlab.Issue, *gitlab.Issue, error) {
	issueLink, _, err := client.IssueLinks.CreateIssueLink(projectID, issueIDD, opts)
	if err != nil {
		return nil, nil, err
	}

	return issueLink.SourceIssue, issueLink.TargetIssue, nil
}

var SetIssueTimeEstimate = func(client *gitlab.Client, projectID any, issueIDD int, opts *gitlab.SetTimeEstimateOptions) (*gitlab.TimeStats, error) {
	timeStats, _, err := client.Issues.SetTimeEstimate(projectID, issueIDD, opts)
	if err != nil {
		return nil, err
	}

	return timeStats, nil
}

var AddIssueTimeSpent = func(client *gitlab.Client, projectID any, issueIDD int, opts *gitlab.AddSpentTimeOptions) (*gitlab.TimeStats, error) {
	timeStats, _, err := client.Issues.AddSpentTime(projectID, issueIDD, opts)
	if err != nil {
		return nil, err
	}

	return timeStats, nil
}
