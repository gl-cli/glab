package api

import (
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

type CreatePushMirrorOptions struct {
	Url                   string
	Enabled               bool
	OnlyProtectedBranches bool
	KeepDivergentRefs     bool
}

type CreatePullMirrorOptions struct {
	Url                   string
	Enabled               bool
	OnlyProtectedBranches bool
}

var CreatePushMirror = func(
	client *gitlab.Client,
	projectID any,
	opts *CreatePushMirrorOptions,
) (*gitlab.ProjectMirror, error) {
	if client == nil {
		client = apiClient.Lab()
	}
	opt := &gitlab.AddProjectMirrorOptions{
		URL:                   &opts.Url,
		Enabled:               &opts.Enabled,
		OnlyProtectedBranches: &opts.OnlyProtectedBranches,
		KeepDivergentRefs:     &opts.KeepDivergentRefs,
	}
	pm, _, err := client.ProjectMirrors.AddProjectMirror(projectID, opt)
	return pm, err
}

var CreatePullMirror = func(
	client *gitlab.Client,
	projectID any,
	opts *CreatePullMirrorOptions,
) error {
	if client == nil {
		client = apiClient.Lab()
	}
	opt := &gitlab.EditProjectOptions{
		ImportURL:                   &opts.Url,
		Mirror:                      &opts.Enabled,
		OnlyMirrorProtectedBranches: &opts.OnlyProtectedBranches,
	}
	_, _, err := client.Projects.EditProject(projectID, opt)
	return err
}
