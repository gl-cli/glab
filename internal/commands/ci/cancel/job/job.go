package job

import (
	"fmt"
	"io"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	gitlab "gitlab.com/gitlab-org/api/client-go"

	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/commands/ci/ciutils"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/internal/iostreams"
)

const (
	FlagDryRun = "dry-run"
)

func NewCmdCancel(f cmdutils.Factory) *cobra.Command {
	jobCancelCmd := &cobra.Command{
		Use:   "job <id> [flags]",
		Short: `Cancel CI/CD jobs.`,
		Example: heredoc.Doc(`
			$ glab ci cancel job 1504182795
			$ glab ci cancel job 1504182795,1504182795
			$ glab ci cancel job "1504182795 1504182795"
			$ glab ci cancel job 1504182795,1504182795 --dry-run
		`),
		Long: ``,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("You must pass a job ID.")
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			c := f.IO().Color()
			apiClient, err := f.HttpClient()
			if err != nil {
				return err
			}

			repo, err := f.BaseRepo()
			if err != nil {
				return err
			}
			dryRunMode, _ := cmd.Flags().GetBool(FlagDryRun)

			var jobIDs []int

			jobIDs, err = ciutils.IDsFromArgs(args)
			if err != nil {
				return err
			}
			return runCancelation(jobIDs, dryRunMode, f.IO().StdOut, c, apiClient, repo)
		},
	}

	SetupCommandFlags(jobCancelCmd.Flags())
	return jobCancelCmd
}

func SetupCommandFlags(flags *pflag.FlagSet) {
	flags.BoolP(FlagDryRun, "", false, "Simulates process, but does not cancel anything.")
}

func runCancelation(
	jobIDs []int,
	dryRunMode bool,
	w io.Writer,
	c *iostreams.ColorPalette,
	apiClient *gitlab.Client,
	repo glrepo.Interface,
) error {
	for _, id := range jobIDs {
		if dryRunMode {
			fmt.Fprintf(w, "%s Job #%d will be canceled.\n", c.DotWarnIcon(), id)
		} else {
			pid, err := repo.Project(apiClient)
			if err != nil {
				return err
			}
			_, _, err = apiClient.Jobs.CancelJob(pid.ID, id)
			if err != nil {
				return err
			}

			fmt.Fprintf(w, "%s Job #%d is canceled successfully.\n", c.RedCheck(), id)
		}
	}
	fmt.Println()

	return nil
}
