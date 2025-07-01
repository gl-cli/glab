// This is a silly wrapper for gitlab client-go but helps maintain consistency
package api

import (
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// ListIssueNotes returns issue notes
// Attention: this is a global variable and may be overridden in tests.
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

// UpdateIssue updates an issue
// Attention: this is a global variable and may be overridden in tests.
var UpdateIssue = func(client *gitlab.Client, projectID any, issueID int, opts *gitlab.UpdateIssueOptions) (*gitlab.Issue, error) {
	issue, _, err := client.Issues.UpdateIssue(projectID, issueID, opts)
	if err != nil {
		return nil, err
	}

	return issue, nil
}

// GetIssue returns an issue
// Attention: this is a global variable and may be overridden in tests.
var GetIssue = func(client *gitlab.Client, projectID any, issueID int) (*gitlab.Issue, error) {
	issue, _, err := client.Issues.GetIssue(projectID, issueID)
	if err != nil {
		return nil, err
	}

	return issue, nil
}

// CreateIssue creates an issue
// Attention: this is a global variable and may be overridden in tests.
var CreateIssue = func(client *gitlab.Client, projectID any, opts *gitlab.CreateIssueOptions) (*gitlab.Issue, error) {
	issue, _, err := client.Issues.CreateIssue(projectID, opts)
	if err != nil {
		return nil, err
	}

	return issue, nil
}

// DeleteIssue deletes an issue
// Attention: this is a global variable and may be overridden in tests.
var DeleteIssue = func(client *gitlab.Client, projectID any, issueID int) error {
	_, err := client.Issues.DeleteIssue(projectID, issueID)
	if err != nil {
		return err
	}

	return nil
}
