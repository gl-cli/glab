package api

import (
	"fmt"
	"net/http"

	"github.com/xanzy/go-gitlab"
)

const (
	commitAuthorName  = "glab"
	commitAuthorEmail = "noreply@glab.gitlab.com"
)

// GetFile retrieves a file from repository. Note that file content is Base64 encoded.
var GetFile = func(client *gitlab.Client, projectID interface{}, path string, ref string) (*gitlab.File, error) {
	if client == nil {
		client = apiClient.Lab()
	}

	fileOpts := &gitlab.GetFileOptions{
		Ref: &ref,
	}
	file, _, err := client.RepositoryFiles.GetFile(projectID, path, fileOpts)
	if err != nil {
		return nil, err
	}

	return file, nil
}

// SyncFile syncs (add or update) a file in the repository
var SyncFile = func(client *gitlab.Client, projectID interface{}, path string, content []byte, ref string) error {
	_, resp, err := client.RepositoryFiles.GetFileMetaData(projectID, path, &gitlab.GetFileMetaDataOptions{
		Ref: gitlab.Ptr(ref),
	})
	if err != nil {
		if resp.StatusCode != http.StatusNotFound {
			return err
		}

		// file does not exist yet, lets create it
		_, _, err := client.RepositoryFiles.CreateFile(projectID, path, &gitlab.CreateFileOptions{
			Branch:        gitlab.Ptr(ref),
			Content:       gitlab.Ptr(string(content)),
			CommitMessage: gitlab.Ptr(fmt.Sprintf("Add %s via glab file sync", path)),
			AuthorName:    gitlab.Ptr(commitAuthorName),
			AuthorEmail:   gitlab.Ptr(commitAuthorEmail),
		})
		return err
	}

	// file already exists, lets update it
	_, _, err = client.RepositoryFiles.UpdateFile(projectID, path, &gitlab.UpdateFileOptions{
		Branch:        gitlab.Ptr(ref),
		Content:       gitlab.Ptr(string(content)),
		CommitMessage: gitlab.Ptr(fmt.Sprintf("Update %s via glab file sync", path)),
		AuthorName:    gitlab.Ptr(commitAuthorName),
		AuthorEmail:   gitlab.Ptr(commitAuthorEmail),
	})
	return err
}

var CreateFile = func(client *gitlab.Client, projectID interface{}, path string, content []byte, ref string) error {
	_, _, err := client.RepositoryFiles.CreateFile(projectID, path, &gitlab.CreateFileOptions{
		Branch:        gitlab.Ptr(ref),
		Content:       gitlab.Ptr(string(content)),
		CommitMessage: gitlab.Ptr(fmt.Sprintf("Add %s via glab file sync", path)),
		AuthorName:    gitlab.Ptr(commitAuthorName),
		AuthorEmail:   gitlab.Ptr(commitAuthorEmail),
	})
	return err
}

var UpdateFile = func(client *gitlab.Client, projectID interface{}, path string, content []byte, ref string) error {
	_, _, err := client.RepositoryFiles.UpdateFile(projectID, path, &gitlab.UpdateFileOptions{
		Branch:        gitlab.Ptr(ref),
		Content:       gitlab.Ptr(string(content)),
		CommitMessage: gitlab.Ptr(fmt.Sprintf("Update %s via glab file sync", path)),
		AuthorName:    gitlab.Ptr(commitAuthorName),
		AuthorEmail:   gitlab.Ptr(commitAuthorEmail),
	})
	return err
}
