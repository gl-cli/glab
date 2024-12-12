package add

import (
	"time"

	gitlab "gitlab.com/gitlab-org/api/client-go"
)

func UploadSSHKey(client *gitlab.Client, title, key, expiresAt string) error {
	sshKeyAddOptions := &gitlab.AddSSHKeyOptions{
		Title: &title,
		Key:   &key,
	}

	if expiresAt != "" {
		expiresAt, err := time.Parse(time.RFC3339[:len(expiresAt)], expiresAt)
		if err != nil {
			return err
		}
		sshKeyAddOptions.ExpiresAt = (*gitlab.ISOTime)(&expiresAt)
	}

	_, _, err := client.Users.AddSSHKey(sshKeyAddOptions)
	return err
}
