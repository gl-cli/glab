package api

import (
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

var ListSecureFiles = func(client *gitlab.Client, l *gitlab.ListProjectSecureFilesOptions, projectID interface{}) ([]*gitlab.SecureFile, error) {
	if client == nil {
		client = apiClient.Lab()
	}

	if l == nil {
		l = &gitlab.ListProjectSecureFilesOptions{
			Page:    1,
			PerPage: DefaultListLimit,
		}
	} else {
		if l.PerPage == 0 {
			l.PerPage = DefaultListLimit
		}
		if l.Page == 0 {
			l.Page = 1
		}
	}

	files, _, err := client.SecureFiles.ListProjectSecureFiles(projectID, l)
	if err != nil {
		return nil, err
	}
	return files, nil
}
