package checkout

import (
	"fmt"
	"strings"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	"github.com/xanzy/go-gitlab"
	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/commands/mr/mrutils"
	"gitlab.com/gitlab-org/cli/pkg/git"
)

type mrCheckoutConfig struct {
	branch   string
	track    bool
	upstream string
}

var mrCheckoutCfg mrCheckoutConfig

func NewCmdCheckout(f *cmdutils.Factory) *cobra.Command {
	mrCheckoutCmd := &cobra.Command{
		Use:   "checkout [<id> | <branch>]",
		Short: "Check out an open merge request.",
		Long:  ``,
		Example: heredoc.Doc(`
			$ glab mr checkout 1
			$ glab mr checkout branch
			$ glab mr checkout 12 --branch todo-fix
			$ glab mr checkout new-feature --set-upstream-to=upstream/main

			# Uses the checked-out branch
			$ glab mr checkout
		`),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			var upstream string

			if mrCheckoutCfg.upstream != "" {
				upstream = mrCheckoutCfg.upstream

				if val := strings.Split(mrCheckoutCfg.upstream, "/"); len(val) > 1 {
					// Verify that we have the remote set
					repo, err := f.Remotes()
					if err != nil {
						return err
					}
					_, err = repo.FindByName(val[0])
					if err != nil {
						return err
					}
				}
			}

			apiClient, err := f.HttpClient()
			if err != nil {
				return err
			}

			mr, _, err := mrutils.MRFromArgs(f, args, "any")
			if err != nil {
				return err
			}

			if mrCheckoutCfg.branch == "" {
				mrCheckoutCfg.branch = mr.SourceBranch
			}

			var mrRef string
			var mrProject *gitlab.Project

			mrProject, err = api.GetProject(apiClient, mr.SourceProjectID)
			if err != nil {
				// If we don't have access to the source project, let's try the target project
				mrProject, err = api.GetProject(apiClient, mr.TargetProjectID)
				if err != nil {
					return err
				} else {
					// We found the target project, let's find the ref another way
					mrRef = fmt.Sprintf("refs/merge-requests/%d/head", mr.IID)
				}

			} else {
				mrRef = fmt.Sprintf("refs/heads/%s", mr.SourceBranch)
			}

			fetchRefSpec := fmt.Sprintf("%s:%s", mrRef, mrCheckoutCfg.branch)
			if err := git.RunCmd([]string{"fetch", mrProject.SSHURLToRepo, fetchRefSpec}); err != nil {
				return err
			}

			// .remote is needed for `git pull` to work
			// .pushRemote is needed for `git push` to work, if user has set `remote.pushDefault`.
			// see https://git-scm.com/docs/git-config#Documentation/git-config.txt-branchltnamegtremote
			if err := git.RunCmd([]string{"config", fmt.Sprintf("branch.%s.remote", mrCheckoutCfg.branch), mrProject.SSHURLToRepo}); err != nil {
				return err
			}
			if mr.AllowCollaboration {
				if err := git.RunCmd([]string{"config", fmt.Sprintf("branch.%s.pushRemote", mrCheckoutCfg.branch), mrProject.SSHURLToRepo}); err != nil {
					return err
				}
			}
			if err := git.RunCmd([]string{"config", fmt.Sprintf("branch.%s.merge", mrCheckoutCfg.branch), mrRef}); err != nil {
				return err
			}

			// Check out branch
			if err := git.CheckoutBranch(mrCheckoutCfg.branch); err != nil {
				return err
			}

			// Check out the branch
			if upstream != "" {
				if err := git.RunCmd([]string{"branch", "--set-upstream-to", upstream}); err != nil {
					return err
				}
			}
			return nil
		},
	}
	mrCheckoutCmd.Flags().StringVarP(&mrCheckoutCfg.branch, "branch", "b", "", "Check out merge request with name <branch>.")
	mrCheckoutCmd.Flags().BoolVarP(&mrCheckoutCfg.track, "track", "t", true, "Set checked out branch to track the remote branch.")
	_ = mrCheckoutCmd.Flags().MarkDeprecated("track", "Now enabled by default")
	mrCheckoutCmd.Flags().StringVarP(&mrCheckoutCfg.upstream, "set-upstream-to", "u", "", "Set tracking of checked-out branch to [REMOTE/]BRANCH.")
	return mrCheckoutCmd
}
