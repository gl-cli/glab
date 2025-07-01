package api

import gitlab "gitlab.com/gitlab-org/api/client-go"

type ListLabelsOptions struct {
	WithCounts *bool
	PerPage    int
	Page       int
}

func (opts *ListLabelsOptions) ListLabelsOptions() *gitlab.ListLabelsOptions {
	projectOpts := &gitlab.ListLabelsOptions{}
	projectOpts.WithCounts = opts.WithCounts
	projectOpts.PerPage = opts.PerPage
	projectOpts.Page = opts.Page
	return projectOpts
}

func (opts *ListLabelsOptions) ListGroupLabelsOptions() *gitlab.ListGroupLabelsOptions {
	groupOpts := &gitlab.ListGroupLabelsOptions{}
	groupOpts.WithCounts = opts.WithCounts
	groupOpts.PerPage = opts.PerPage
	groupOpts.Page = opts.Page
	return groupOpts
}

// ListLabels returns a list of project labels
// Attention: this is a global variable and may be overridden in tests.
var ListLabels = func(client *gitlab.Client, projectID any, opts *ListLabelsOptions) ([]*gitlab.Label, error) {
	if opts.PerPage == 0 {
		opts.PerPage = DefaultListLimit
	}

	label, _, err := client.Labels.ListLabels(projectID, opts.ListLabelsOptions())
	if err != nil {
		return nil, err
	}
	return label, nil
}
