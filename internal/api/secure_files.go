package api

import (
	"io"

	gitlab "gitlab.com/gitlab-org/api/client-go"
)

var CreateSecureFile = func(client *gitlab.Client, projectID any, filename string, content io.Reader) error {
	opts := &gitlab.CreateSecureFileOptions{
		Name: &filename,
	}
	_, _, err := client.SecureFiles.CreateSecureFile(projectID, content, opts)
	return err
}

var DownloadSecureFile = func(client *gitlab.Client, projectID any, id int) (io.Reader, error) {
	reader, _, err := client.SecureFiles.DownloadSecureFile(projectID, id)
	if err != nil {
		return nil, err
	}
	return reader, nil
}

var GetSecureFile = func(client *gitlab.Client, projectID any, id int) (*gitlab.SecureFile, error) {
	file, _, err := client.SecureFiles.ShowSecureFileDetails(projectID, id)
	return file, err
}

var ListSecureFiles = func(client *gitlab.Client, l *gitlab.ListProjectSecureFilesOptions, projectID any) ([]*gitlab.SecureFile, error) {
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

var RemoveSecureFile = func(client *gitlab.Client, projectID any, id int) error {
	_, err := client.SecureFiles.RemoveSecureFile(projectID, id)
	return err
}
