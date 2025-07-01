package api

import (
	"fmt"

	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// CurrentUser returns the user for the API token
// Attention: this is a global variable and may be overridden in tests.
var CurrentUser = func(client *gitlab.Client) (*gitlab.User, error) {
	u, _, err := client.Users.CurrentUser()
	if err != nil {
		return nil, err
	}
	return u, nil
}

func UserByName(client *gitlab.Client, name string) (*gitlab.User, error) {
	opts := &gitlab.ListUsersOptions{Username: gitlab.Ptr(name)}

	if opts.PerPage == 0 {
		opts.PerPage = DefaultListLimit
	}

	// Handle special case of '@me' which maps to the currently authenticated user
	if name == "@me" {
		u, _, err := client.Users.CurrentUser()
		return u, err
	}

	users, _, err := client.Users.ListUsers(opts)
	if err != nil {
		return nil, err
	}

	if len(users) != 1 {
		return nil, fmt.Errorf("failed to find user by name: %s", name)
	}

	return users[0], nil
}

var UsersByNames = func(client *gitlab.Client, names []string) ([]*gitlab.User, error) {
	users := make([]*gitlab.User, 0, len(names))
	for _, name := range names {
		user, err := UserByName(client, name)
		if err != nil {
			return nil, err
		}

		users = append(users, user)
	}
	return users, nil
}
