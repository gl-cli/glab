package api

import gitlab "gitlab.com/gitlab-org/api/client-go"

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

var DeleteProject = func(client *gitlab.Client, projectID any) (*gitlab.Response, error) {
	project, err := client.Projects.DeleteProject(projectID, nil)
	if err != nil {
		return nil, err
	}
	return project, nil
}

var CreateProject = func(client *gitlab.Client, opts *gitlab.CreateProjectOptions) (*gitlab.Project, error) {
	project, _, err := client.Projects.CreateProject(opts)
	if err != nil {
		return nil, err
	}
	return project, nil
}

var UpdateProject = func(client *gitlab.Client, projectID any, opts *gitlab.EditProjectOptions) (*gitlab.Project, error) {
	project, _, err := client.Projects.EditProject(projectID, opts)
	if err != nil {
		return nil, err
	}
	return project, nil
}

var ForkProject = func(client *gitlab.Client, projectID any, opts *gitlab.ForkProjectOptions) (*gitlab.Project, error) {
	project, _, err := client.Projects.ForkProject(projectID, opts)
	if err != nil {
		return nil, err
	}
	return project, nil
}

var GetGroup = func(client *gitlab.Client, groupID any) (*gitlab.Group, error) {
	group, _, err := client.Groups.GetGroup(groupID, &gitlab.GetGroupOptions{})
	if err != nil {
		return nil, err
	}
	return group, nil
}

var ListGroupProjects = func(client *gitlab.Client, groupID any, opts *gitlab.ListGroupProjectsOptions) ([]*gitlab.Project, *gitlab.Response, error) {
	project, resp, err := client.Groups.ListGroupProjects(groupID, opts)
	if err != nil {
		return nil, nil, err
	}
	return project, resp, nil
}

var ListProjectsGroups = func(client *gitlab.Client, projectID any, opts *gitlab.ListProjectGroupOptions) ([]*gitlab.ProjectGroup, error) {
	groups, _, err := client.Projects.ListProjectsGroups(projectID, opts)
	if err != nil {
		return nil, err
	}
	return groups, nil
}

var ListProjectMembers = func(client *gitlab.Client, projectID any, opts *gitlab.ListProjectMembersOptions) ([]*gitlab.ProjectMember, error) {
	members, _, err := client.ProjectMembers.ListAllProjectMembers(projectID, opts)
	if err != nil {
		return nil, err
	}
	return members, nil
}
