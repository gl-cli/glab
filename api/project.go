package api

import gitlab "gitlab.com/gitlab-org/api/client-go"

var GetProject = func(client *gitlab.Client, projectID any) (*gitlab.Project, error) {
	if client == nil {
		client = apiClient.Lab()
	}
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

var DeleteProject = func(client *gitlab.Client, projectID any) (*gitlab.Response, error) {
	if client == nil {
		client = apiClient.Lab()
	}
	project, err := client.Projects.DeleteProject(projectID, nil)
	if err != nil {
		return nil, err
	}
	return project, nil
}

var CreateProject = func(client *gitlab.Client, opts *gitlab.CreateProjectOptions) (*gitlab.Project, error) {
	if client == nil {
		client = apiClient.Lab()
	}
	project, _, err := client.Projects.CreateProject(opts)
	if err != nil {
		return nil, err
	}
	return project, nil
}

var ForkProject = func(client *gitlab.Client, projectID any, opts *gitlab.ForkProjectOptions) (*gitlab.Project, error) {
	if client == nil {
		client = apiClient.Lab()
	}
	project, _, err := client.Projects.ForkProject(projectID, opts)
	if err != nil {
		return nil, err
	}
	return project, nil
}

var GetGroup = func(client *gitlab.Client, groupID any) (*gitlab.Group, error) {
	if client == nil {
		client = apiClient.Lab()
	}
	group, _, err := client.Groups.GetGroup(groupID, &gitlab.GetGroupOptions{})
	if err != nil {
		return nil, err
	}
	return group, nil
}

var ListGroupProjects = func(client *gitlab.Client, groupID any, opts *gitlab.ListGroupProjectsOptions) ([]*gitlab.Project, *gitlab.Response, error) {
	if client == nil {
		client = apiClient.Lab()
	}
	project, resp, err := client.Groups.ListGroupProjects(groupID, opts)
	if err != nil {
		return nil, nil, err
	}
	return project, resp, nil
}

var ListProjectsGroups = func(client *gitlab.Client, projectID any, opts *gitlab.ListProjectGroupOptions) ([]*gitlab.ProjectGroup, error) {
	if client == nil {
		client = apiClient.Lab()
	}
	groups, _, err := client.Projects.ListProjectsGroups(projectID, opts)
	if err != nil {
		return nil, err
	}
	return groups, nil
}

var ListProjectMembers = func(client *gitlab.Client, projectID any, opts *gitlab.ListProjectMembersOptions) ([]*gitlab.ProjectMember, error) {
	if client == nil {
		client = apiClient.Lab()
	}
	members, _, err := client.ProjectMembers.ListAllProjectMembers(projectID, opts)
	if err != nil {
		return nil, err
	}
	return members, nil
}
