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

func getClient(client *gitlab.Client) *gitlab.Client {
	if client == nil {
		return apiClient.Lab()
	}
	return client
}

var CreateLabel = func(client *gitlab.Client, projectID interface{}, opts *gitlab.CreateLabelOptions) (*gitlab.Label, error) {
	client = getClient(client)

	label, _, err := client.Labels.CreateLabel(projectID, opts)
	if err != nil {
		return nil, err
	}
	return label, nil
}

var ListLabels = func(client *gitlab.Client, projectID interface{}, opts *ListLabelsOptions) ([]*gitlab.Label, error) {
	client = getClient(client)

	if opts.PerPage == 0 {
		opts.PerPage = DefaultListLimit
	}

	label, _, err := client.Labels.ListLabels(projectID, opts.ListLabelsOptions())
	if err != nil {
		return nil, err
	}
	return label, nil
}

var ListGroupLabels = func(client *gitlab.Client, groupID interface{}, opts *ListLabelsOptions) ([]*gitlab.GroupLabel, error) {
	client = getClient(client)

	if opts.PerPage == 0 {
		opts.PerPage = DefaultListLimit
	}

	labels, _, err := client.GroupLabels.ListGroupLabels(groupID, opts.ListGroupLabelsOptions())
	if err != nil {
		return nil, err
	}
	return labels, nil
}
