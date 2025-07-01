package get

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
)

func NewCmdGet(f cmdutils.Factory) *cobra.Command {
	securefileGetCmd := &cobra.Command{
		Use:     "get <fileID>",
		Short:   `Get details of a project secure file. (GitLab 18.0 and later)`,
		Long:    ``,
		Aliases: []string{"show"},
		Args:    cobra.ExactArgs(1),
		Example: heredoc.Doc(`
			# Get details of a project's secure file using the file ID.
			$ glab securefile get 1

			# Get details of a project's secure file using the 'show' alias.
			$ glab securefile show 1
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			apiClient, err := f.HttpClient()
			if err != nil {
				return err
			}

			repo, err := f.BaseRepo()
			if err != nil {
				return err
			}

			fileID, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("Secure file ID must be an integer: %s", args[0])
			}

			file, _, err := apiClient.SecureFiles.ShowSecureFileDetails(repo.FullName(), fileID)
			if err != nil {
				return fmt.Errorf("Error getting secure file: %v", err)
			}

			fileJSON, _ := json.Marshal(file)
			fmt.Fprintln(f.IO().StdOut, string(fileJSON))
			return nil
		},
	}

	return securefileGetCmd
}
