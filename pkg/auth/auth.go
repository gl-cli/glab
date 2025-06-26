package auth

import (
	"fmt"

	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
)

func GetAuthenticatedClient(f cmdutils.Factory) (*gitlab.Client, error) {
	glConfig := f.Config()

	instances, err := glConfig.Hosts()
	if err != nil || len(instances) == 0 {
		return nil, fmt.Errorf("no GitLab instances have been authenticated with glab. Run `%s` to authenticate.", f.IO().Color().Bold("glab auth login"))
	}

	labClient, err := f.HttpClient()
	if err != nil {
		return nil, fmt.Errorf("error using API client: %v", err)
	}

	return labClient, nil
}
