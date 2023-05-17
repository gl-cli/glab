package api

import "github.com/xanzy/go-gitlab"

var GenerateChangelog = func(client *gitlab.Client, projectID interface{}, options *gitlab.GenerateChangelogDataOptions) (*gitlab.ChangelogData, error) {
	if client == nil {
		client = apiClient.Lab()
	}

	changelog, _, err := client.Repositories.GenerateChangelogData(projectID, *options)
	if err != nil {
		return nil, err
	}

	return changelog, nil
}
