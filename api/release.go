package api

import gitlab "gitlab.com/gitlab-org/api/client-go"

var CreateRelease = func(client *gitlab.Client, projectID any, opts *gitlab.CreateReleaseOptions) (*gitlab.Release, error) {
	release, _, err := client.Releases.CreateRelease(projectID, opts)
	if err != nil {
		return nil, err
	}

	return release, nil
}

var GetRelease = func(client *gitlab.Client, projectID any, tag string) (*gitlab.Release, error) {
	release, _, err := client.Releases.GetRelease(projectID, tag)
	if err != nil {
		return nil, err
	}

	return release, nil
}

var ListReleases = func(client *gitlab.Client, projectID any, opts *gitlab.ListReleasesOptions) ([]*gitlab.Release, error) {
	if opts.PerPage == 0 {
		opts.PerPage = DefaultListLimit
	}

	releases, _, err := client.Releases.ListReleases(projectID, opts)
	if err != nil {
		return nil, err
	}

	return releases, nil
}
