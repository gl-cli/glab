package api

import gitlab "gitlab.com/gitlab-org/api/client-go"

// GetRelease returns a single release
// Attention: this is a global variable and may be overridden in tests.
var GetRelease = func(client *gitlab.Client, projectID any, tag string) (*gitlab.Release, error) {
	release, _, err := client.Releases.GetRelease(projectID, tag)
	if err != nil {
		return nil, err
	}

	return release, nil
}

// ListReleases list all releases of a project
// Attention: this is a global variable and may be overridden in tests.
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
