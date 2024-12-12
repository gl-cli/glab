package api

import (
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

var CurrentUserEvents = func(client *gitlab.Client, opts *gitlab.ListContributionEventsOptions) ([]*gitlab.ContributionEvent, error) {
	if client == nil {
		client = apiClient.Lab()
	}

	if opts.PerPage == 0 {
		opts.PerPage = DefaultListLimit
	}

	events, _, err := client.Events.ListCurrentUserContributionEvents(opts)
	if err != nil {
		return nil, err
	}
	return events, nil
}
