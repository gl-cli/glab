package api

import (
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

var ListProjectAccessTokens = func(client *gitlab.Client, projectID any, opts *gitlab.ListProjectAccessTokensOptions) ([]*gitlab.ProjectAccessToken, error) {
	perPage := opts.PerPage
	if perPage == 0 {
		perPage = 100
	}
	tokens := make([]*gitlab.ProjectAccessToken, 0, perPage)
	for {
		results, response, err := client.ProjectAccessTokens.ListProjectAccessTokens(projectID, opts)
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, results...)

		if response.CurrentPage >= response.TotalPages {
			break
		}
		opts.Page = response.NextPage
	}

	return tokens, nil
}

var ListGroupAccessTokens = func(client *gitlab.Client, groupID any, opts *gitlab.ListGroupAccessTokensOptions) ([]*gitlab.GroupAccessToken, error) {
	perPage := opts.PerPage
	if perPage == 0 {
		perPage = 100
	}
	tokens := make([]*gitlab.GroupAccessToken, 0, perPage)
	for {
		results, response, err := client.GroupAccessTokens.ListGroupAccessTokens(groupID, opts)
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, results...)

		if response.CurrentPage >= response.TotalPages {
			break
		}
		opts.Page = response.NextPage
	}

	return tokens, nil
}

var ListPersonalAccessTokens = func(client *gitlab.Client, opts *gitlab.ListPersonalAccessTokensOptions) ([]*gitlab.PersonalAccessToken, error) {
	perPage := opts.PerPage
	if perPage == 0 {
		perPage = 100
	}
	tokens := make([]*gitlab.PersonalAccessToken, 0, perPage)
	for {
		results, response, err := client.PersonalAccessTokens.ListPersonalAccessTokens(opts)
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, results...)

		if response.CurrentPage >= response.TotalPages {
			break
		}
		opts.Page = response.NextPage
	}

	return tokens, nil
}

var CreateProjectAccessToken = func(client *gitlab.Client, pid any, opts *gitlab.CreateProjectAccessTokenOptions) (*gitlab.ProjectAccessToken, error) {
	token, _, err := client.ProjectAccessTokens.CreateProjectAccessToken(pid, opts)
	return token, err
}

var CreateGroupAccessToken = func(client *gitlab.Client, gid any, opts *gitlab.CreateGroupAccessTokenOptions) (*gitlab.GroupAccessToken, error) {
	token, _, err := client.GroupAccessTokens.CreateGroupAccessToken(gid, opts)
	return token, err
}

var CreatePersonalAccessToken = func(client *gitlab.Client, uid int, opts *gitlab.CreatePersonalAccessTokenOptions) (*gitlab.PersonalAccessToken, error) {
	token, _, err := client.Users.CreatePersonalAccessToken(uid, opts)
	return token, err
}

var RevokeProjectAccessToken = func(client *gitlab.Client, pid any, id int) error {
	_, err := client.ProjectAccessTokens.RevokeProjectAccessToken(pid, id)
	return err
}

var RevokeGroupAccessToken = func(client *gitlab.Client, gid any, id int) error {
	_, err := client.GroupAccessTokens.RevokeGroupAccessToken(gid, id)
	return err
}

var RevokePersonalAccessToken = func(client *gitlab.Client, id int) error {
	_, err := client.PersonalAccessTokens.RevokePersonalAccessToken(id)
	return err
}

var RotateProjectAccessToken = func(client *gitlab.Client, pid any, id int, opts *gitlab.RotateProjectAccessTokenOptions) (*gitlab.ProjectAccessToken, error) {
	token, _, err := client.ProjectAccessTokens.RotateProjectAccessToken(pid, id, opts)
	return token, err
}

var RotateGroupAccessToken = func(client *gitlab.Client, gid any, id int, opts *gitlab.RotateGroupAccessTokenOptions) (*gitlab.GroupAccessToken, error) {
	token, _, err := client.GroupAccessTokens.RotateGroupAccessToken(gid, id, opts)
	return token, err
}

var RotatePersonalAccessToken = func(client *gitlab.Client, id int, opts *gitlab.RotatePersonalAccessTokenOptions) (*gitlab.PersonalAccessToken, error) {
	token, _, err := client.PersonalAccessTokens.RotatePersonalAccessToken(id, opts)
	return token, err
}
