package api

import gitlab "gitlab.com/gitlab-org/api/client-go"

// GetProject returns a single project
// Attention: this is a global variable and may be overridden in tests.
var GetProject = func(client *gitlab.Client, projectID any) (*gitlab.Project, error) {
	opts := &gitlab.GetProjectOptions{
		License:              gitlab.Ptr(true),
		WithCustomAttributes: gitlab.Ptr(true),
	}
	project, _, err := client.Projects.GetProject(projectID, opts)
	if err != nil {
		return nil, err
	}
	return project, nil
}

// CreateProject creates a new project
// Attention: this is a global variable and may be overridden in tests.
var CreateProject = func(client *gitlab.Client, opts *gitlab.CreateProjectOptions) (*gitlab.Project, error) {
	project, _, err := client.Projects.CreateProject(opts)
	if err != nil {
		return nil, err
	}
	return project, nil
}

// ListProjectMembers lists all members of a project
// Attention: this is a global variable and may be overridden in tests.
var ListProjectMembers = func(client *gitlab.Client, projectID any, opts *gitlab.ListProjectMembersOptions) ([]*gitlab.ProjectMember, error) {
	members, _, err := client.ProjectMembers.ListAllProjectMembers(projectID, opts)
	if err != nil {
		return nil, err
	}
	return members, nil
}
