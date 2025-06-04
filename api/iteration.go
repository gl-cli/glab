package api

import gitlab "gitlab.com/gitlab-org/api/client-go"

type ListProjectIterationsOptions struct {
	IncludeAncestors *bool
	PerPage          int
	Page             int
}

func (opts *ListProjectIterationsOptions) ListProjectIterationsOptions() *gitlab.ListProjectIterationsOptions {
	projectOpts := &gitlab.ListProjectIterationsOptions{}
	projectOpts.IncludeAncestors = opts.IncludeAncestors
	projectOpts.PerPage = opts.PerPage
	projectOpts.Page = opts.Page
	return projectOpts
}

func (opts *ListProjectIterationsOptions) ListGroupIterationsOptions() *gitlab.ListGroupIterationsOptions {
	groupOpts := &gitlab.ListGroupIterationsOptions{}
	groupOpts.IncludeAncestors = opts.IncludeAncestors
	groupOpts.PerPage = opts.PerPage
	groupOpts.Page = opts.Page
	return groupOpts
}

var ListProjectIterations = func(client *gitlab.Client, projectID interface{}, opts *ListProjectIterationsOptions) ([]*gitlab.ProjectIteration, error) {
	if opts.PerPage == 0 {
		opts.PerPage = DefaultListLimit
	}

	iteration, _, err := client.ProjectIterations.ListProjectIterations(projectID, opts.ListProjectIterationsOptions())
	if err != nil {
		return nil, err
	}
	return iteration, nil
}

var ListGroupIterations = func(client *gitlab.Client, groupID interface{}, opts *ListProjectIterationsOptions) ([]*gitlab.GroupIteration, error) {
	if opts.PerPage == 0 {
		opts.PerPage = DefaultListLimit
	}

	iterations, _, err := client.GroupIterations.ListGroupIterations(groupID, opts.ListGroupIterationsOptions())
	if err != nil {
		return nil, err
	}
	return iterations, nil
}
