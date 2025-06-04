package api

import gitlab "gitlab.com/gitlab-org/api/client-go"

var GetMetadata = func(client *gitlab.Client) (*gitlab.Metadata, error) {
	md, _, err := client.Metadata.GetMetadata()
	if err != nil {
		return nil, err
	}

	return md, nil
}
