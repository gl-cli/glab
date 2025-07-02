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
