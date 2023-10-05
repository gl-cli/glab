package flag

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/cli/commands/cmdtest"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
)

type ListOptions struct {
	Group    string
	BaseRepo func() (glrepo.Interface, error)
}

func NewDummyCmd(f *cmdutils.Factory, runE func(opts *ListOptions) error) *cobra.Command {
	opts := &ListOptions{}

	cmd := &cobra.Command{
		Use:   "list [flags]",
		Short: "List objects",
		Long:  "List objects",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			// support repo override
			opts.BaseRepo = f.BaseRepo

			group, err := GroupOverride(cmd)
			if err != nil {
				return err
			}
			opts.Group = group

			if runE != nil {
				return runE(opts)
			}

			return nil
		},
	}
	cmdutils.EnableRepoOverride(cmd, f)
	cmd.PersistentFlags().StringP("group", "g", "", "Select another group or it's subgroups")

	return cmd
}

func runCommand(cli string, runE func(opts *ListOptions) error, doHyperlinks string) error {
	factory := &cmdutils.Factory{
		Config: func() (config.Config, error) {
			return config.NewBlankConfig(), nil
		},
		BaseRepo: func() (glrepo.Interface, error) {
			return glrepo.New("OWNER", "REPO"), nil
		},
	}

	cmd := NewDummyCmd(factory, runE)

	_, err := cmdtest.ExecuteCommand(cmd, cli, nil, nil)
	return err
}

func TestGroupOverride(t *testing.T) {
	// set GITLAB_GROUP environment variable
	t.Setenv("GITLAB_GROUP", "GROUP/NAME")

	t.Run("List command with GITLAB_GROUP env var and group", func(t *testing.T) {
		gotOpts := &ListOptions{}

		err := runCommand("--group GROUP/OVERRIDE", func(opts *ListOptions) error {
			gotOpts = opts
			return nil
		}, "")

		assert.NoError(t, err)
		gotGroup := gotOpts.Group
		// make sure the GITLAB_GROUP env variable is not used
		assert.NotEqual(t, gotGroup, "GROUP/NAME")
		// make sure the group flag was used instead
		assert.Equal(t, gotGroup, "GROUP/OVERRIDE")
	})

	t.Run("List command with GITLAB_GROUP env var and no group flag", func(t *testing.T) {
		gotOpts := &ListOptions{}

		err := runCommand("", func(opts *ListOptions) error {
			gotOpts = opts
			return nil
		}, "")

		assert.NoError(t, err)
		gotGroup := gotOpts.Group
		// make sure the GITLAB_GROUP env variable is used
		assert.Equal(t, gotGroup, "GROUP/NAME")
	})

	t.Run("List command with GITLAB_GROUP env var and repo flag", func(t *testing.T) {
		gotOpts := &ListOptions{}

		err := runCommand("--repo OWNER2/REPO2", func(opts *ListOptions) error {
			gotOpts = opts
			return nil
		}, "")

		assert.NoError(t, err)
		gotGroup := gotOpts.Group
		// make sure the GITLAB_GROUP env variable is not used
		assert.Equal(t, gotGroup, "")
		// make sure the repo option is used instead
		gotRepo, _ := gotOpts.BaseRepo()
		assert.Equal(t, gotRepo.FullName(), "OWNER2/REPO2")
	})

	t.Run("List command with GITLAB_GROUP env var and base repo but no repo flag", func(t *testing.T) {
		gotOpts := &ListOptions{}

		err := runCommand("", func(opts *ListOptions) error {
			gotOpts = opts
			return nil
		}, "")

		assert.NoError(t, err)
		gotGroup := gotOpts.Group
		// make sure the GITLAB_GROUP env variable is used
		assert.Equal(t, gotGroup, "GROUP/NAME")
		// make sure the default baserepo is still used
		gotRepo, _ := gotOpts.BaseRepo()
		assert.Equal(t, gotRepo.FullName(), "OWNER/REPO")
	})

	t.Run("List command with GITLAB_GROUP env var, repo and groups flags", func(t *testing.T) {
		gotOpts := &ListOptions{}

		err := runCommand("--repo OWNER2/REPO2 --group GROUP/OVERRIDE", func(opts *ListOptions) error {
			gotOpts = opts
			return nil
		}, "")

		assert.NoError(t, err)
		gotGroup := gotOpts.Group
		// make sure no group option is set at all (group settings ignored)
		assert.Equal(t, gotGroup, "")
		// make sure the repo flag is used instead
		gotRepo, _ := gotOpts.BaseRepo()
		assert.Equal(t, gotRepo.FullName(), "OWNER2/REPO2")
	})
}
