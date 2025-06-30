package transfer

import (
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
)

func NewCmdTransfer(f cmdutils.Factory) *cobra.Command {
	repoTransferCmd := &cobra.Command{
		Use:   "transfer [repo] [flags]",
		Short: `Transfer a repository to a new namespace.`,
		Example: heredoc.Doc(`
			$ glab repo transfer profclems/glab --target-namespace notprofclems
		`),
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error

			var gitlabClient *gitlab.Client
			var repo glrepo.Interface
			if len(args) != 0 {
				// repository is coming from command args, not -R
				repo, err = glrepo.FromFullName(args[0], f.DefaultHostname())
				if err != nil {
					return err
				}

				apiClient, err := f.ApiClient(repo.RepoHost(), f.Config())
				if err != nil {
					return err
				}

				gitlabClient = apiClient.Lab()
			} else {
				gitlabClient, err = f.HttpClient()
				if err != nil {
					return err
				}
				repo, err = f.BaseRepo()
				if err != nil {
					return err
				}
			}

			dontPromptForConfirmation, err := cmd.Flags().GetBool("yes")
			if err != nil {
				return err
			}

			targetNamespace, err := cmd.Flags().GetString("target-namespace")
			if err != nil {
				return err
			}

			c := f.IO().Color()
			fmt.Printf(heredoc.Doc(`
				🔴 WARNING: This operation can be irreversible! 🔴

				If you don't have access to the target namespace:

				- You will lose control of the repository.
				- You won't be able to transfer the repository back to the original namespace, UNLESS you have administrative access
				to the target namespace.

				Source repository: %s
				Target namespace: %s

			`), c.Yellow(repo.FullName()), c.Yellow(targetNamespace))

			if !dontPromptForConfirmation {
				err = cmdutils.ConfirmTransfer()
				if err != nil {
					return fmt.Errorf("unable to confirm: %w", err)
				}
			}

			opt := &gitlab.TransferProjectOptions{}
			opt.Namespace = targetNamespace

			project, _, err := gitlabClient.Projects.TransferProject(repo.FullName(), opt)
			if err != nil {
				return err
			}

			fmt.Fprintf(f.IO().StdOut, "%s Successfully transferred repository %s to %s.\n",
				c.GreenCheck(), c.Yellow(repo.FullName()), c.Yellow(project.PathWithNamespace))

			return nil
		},
	}

	repoTransferCmd.Flags().BoolP("yes", "y", false, "Warning: Skip confirmation prompt and force transfer operation. Transfer cannot be undone.")
	repoTransferCmd.Flags().StringP("target-namespace", "t", "", "The namespace where your project should be transferred to.")

	_ = repoTransferCmd.MarkFlagRequired("target-namespace")

	return repoTransferCmd
}
