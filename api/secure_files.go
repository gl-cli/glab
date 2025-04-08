package api

import (
	"io"

	gitlab "gitlab.com/gitlab-org/api/client-go"
)

var DownloadSecureFile = func(client *gitlab.Client, projectID interface{}, id int) (io.Reader, error) {
	if client == nil {
		client = apiClient.Lab()
	}

	reader, _, err := client.SecureFiles.DownloadSecureFile(projectID, id)
	if err != nil {
		return nil, err
	}
	return reader, nil
}

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
