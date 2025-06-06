package api

import (
	"github.com/hashicorp/go-retryablehttp"

	gitlab "gitlab.com/gitlab-org/api/client-go"
)

var CreateProjectVariable = func(client *gitlab.Client, projectID any, opts *gitlab.CreateProjectVariableOptions) (*gitlab.ProjectVariable, error) {
	vars, _, err := client.ProjectVariables.CreateVariable(projectID, opts)
	if err != nil {
		return nil, err
	}

	return vars, nil
}

var ListProjectVariables = func(client *gitlab.Client, projectID any, opts *gitlab.ListProjectVariablesOptions) ([]*gitlab.ProjectVariable, error) {
	vars, _, err := client.ProjectVariables.ListVariables(projectID, opts)
	if err != nil {
		return nil, err
	}

	return vars, nil
}

var GetProjectVariable = func(client *gitlab.Client, projectID any, key string, scope string) (*gitlab.ProjectVariable, error) {
	reqOpts := &gitlab.GetProjectVariableOptions{
		Filter: &gitlab.VariableFilter{
			EnvironmentScope: scope,
		},
	}
	vars, _, err := client.ProjectVariables.GetVariable(projectID, key, reqOpts)
	if err != nil {
		return nil, err
	}

	return vars, nil
}

var DeleteProjectVariable = func(client *gitlab.Client, projectID any, key string, scope string) error {
	reqOpts := &gitlab.RemoveProjectVariableOptions{
		Filter: &gitlab.VariableFilter{
			EnvironmentScope: scope,
		},
	}
	_, err := client.ProjectVariables.RemoveVariable(projectID, key, reqOpts)
	if err != nil {
		return err
	}

	return nil
}

var UpdateProjectVariable = func(client *gitlab.Client, projectID any, key string, opts *gitlab.UpdateProjectVariableOptions) (*gitlab.ProjectVariable, error) {
	filter := func(request *retryablehttp.Request) error {
		q := request.URL.Query()
		q.Add("filter[environment_scope]", *opts.EnvironmentScope)

		request.URL.RawQuery = q.Encode()

		return nil
	}

	vars, _, err := client.ProjectVariables.UpdateVariable(projectID, key, opts, filter)
	if err != nil {
		return nil, err
	}

	return vars, nil
}

var ListGroupVariables = func(client *gitlab.Client, groupID any, opts *gitlab.ListGroupVariablesOptions) ([]*gitlab.GroupVariable, error) {
	vars, _, err := client.GroupVariables.ListVariables(groupID, opts)
	if err != nil {
		return nil, err
	}

	return vars, nil
}

var CreateGroupVariable = func(client *gitlab.Client, groupID any, opts *gitlab.CreateGroupVariableOptions) (*gitlab.GroupVariable, error) {
	vars, _, err := client.GroupVariables.CreateVariable(groupID, opts)
	if err != nil {
		return nil, err
	}

	return vars, nil
}

var GetGroupVariable = func(client *gitlab.Client, groupID any, key string, scope string) (*gitlab.GroupVariable, error) {
	reqOpts := &gitlab.GetGroupVariableOptions{
		Filter: &gitlab.VariableFilter{
			EnvironmentScope: scope,
		},
	}
	vars, _, err := client.GroupVariables.GetVariable(groupID, key, reqOpts)
	if err != nil {
		return nil, err
	}

	return vars, nil
}

var DeleteGroupVariable = func(client *gitlab.Client, groupID any, key string) error {
	_, err := client.GroupVariables.RemoveVariable(groupID, key, nil)
	if err != nil {
		return err
	}

	return nil
}

var UpdateGroupVariable = func(client *gitlab.Client, groupID any, key string, opts *gitlab.UpdateGroupVariableOptions) (*gitlab.GroupVariable, error) {
	vars, _, err := client.GroupVariables.UpdateVariable(groupID, key, opts)
	if err != nil {
		return nil, err
	}

	return vars, nil
}
