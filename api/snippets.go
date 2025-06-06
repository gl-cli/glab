package api

import (
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// CreateSnippet for the user inside the users snippets
var CreateSnippet = func(
	client *gitlab.Client,
	opts *gitlab.CreateSnippetOptions,
) (*gitlab.Snippet, error) {
	snippet, _, err := client.Snippets.CreateSnippet(opts)
	if err != nil {
		return nil, err
	}
	return snippet, nil
}

// CreateProjectSnippet inside the project
var CreateProjectSnippet = func(
	client *gitlab.Client,
	projectID any,
	opts *gitlab.CreateProjectSnippetOptions,
) (*gitlab.Snippet, error) {
	snippet, _, err := client.ProjectSnippets.CreateSnippet(projectID, opts)
	if err != nil {
		return nil, err
	}

	return snippet, nil
}
