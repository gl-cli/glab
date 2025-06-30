package flag

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/cli/internal/testing/cmdtest"
	"gitlab.com/gitlab-org/cli/internal/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/internal/glinstance"
)

type options struct {
	group    string
	baseRepo func() (glrepo.Interface, error)
}

func NewDummyCmd(f cmdutils.Factory, runE func(opts *options) error) *cobra.Command {
	opts := &options{
		baseRepo: f.BaseRepo,
	}

	cmd := &cobra.Command{
		Use:   "list [flags]",
		Short: "List objects",
		Long:  "List objects",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := opts.complete(cmd); err != nil {
				return err
			}

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

func (o *options) complete(cmd *cobra.Command) error {
	group, err := GroupOverride(cmd)
	if err != nil {
		return err
	}
	o.group = group

	return nil
}

func runCommand(cli string, runE func(opts *options) error, doHyperlinks string) error {
	factory := &cmdtest.Factory{
		ConfigStub: func() config.Config {
			return config.NewBlankConfig()
		},
		BaseRepoStub: func() (glrepo.Interface, error) {
			return glrepo.New("OWNER", "REPO", glinstance.DefaultHostname), nil
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
		gotOpts := &options{}

		err := runCommand("--group GROUP/OVERRIDE", func(opts *options) error {
			gotOpts = opts
			return nil
		}, "")

		assert.NoError(t, err)
		gotGroup := gotOpts.group
		// make sure the GITLAB_GROUP env variable is not used
		assert.NotEqual(t, gotGroup, "GROUP/NAME")
		// make sure the group flag was used instead
		assert.Equal(t, gotGroup, "GROUP/OVERRIDE")
	})

	t.Run("List command with GITLAB_GROUP env var and no group flag", func(t *testing.T) {
		gotOpts := &options{}

		err := runCommand("", func(opts *options) error {
			gotOpts = opts
			return nil
		}, "")

		assert.NoError(t, err)
		gotGroup := gotOpts.group
		// make sure the GITLAB_GROUP env variable is used
		assert.Equal(t, gotGroup, "GROUP/NAME")
	})

	t.Run("List command with GITLAB_GROUP env var and repo flag", func(t *testing.T) {
		gotOpts := &options{}

		err := runCommand("--repo OWNER2/REPO2", func(opts *options) error {
			gotOpts = opts
			return nil
		}, "")

		assert.NoError(t, err)
		gotGroup := gotOpts.group
		// make sure the GITLAB_GROUP env variable is not used
		assert.Equal(t, gotGroup, "")
		// make sure the repo option is used instead
		gotRepo, _ := gotOpts.baseRepo()
		assert.Equal(t, gotRepo.FullName(), "OWNER2/REPO2")
	})

	t.Run("List command with GITLAB_GROUP env var and base repo but no repo flag", func(t *testing.T) {
		gotOpts := &options{}

		err := runCommand("", func(opts *options) error {
			gotOpts = opts
			return nil
		}, "")

		assert.NoError(t, err)
		gotGroup := gotOpts.group
		// make sure the GITLAB_GROUP env variable is used
		assert.Equal(t, gotGroup, "GROUP/NAME")
		// make sure the default baserepo is still used
		gotRepo, _ := gotOpts.baseRepo()
		assert.Equal(t, gotRepo.FullName(), "OWNER/REPO")
	})

	t.Run("List command with GITLAB_GROUP env var, repo and groups flags", func(t *testing.T) {
		gotOpts := &options{}

		err := runCommand("--repo OWNER2/REPO2 --group GROUP/OVERRIDE", func(opts *options) error {
			gotOpts = opts
			return nil
		}, "")

		assert.NoError(t, err)
		gotGroup := gotOpts.group
		// make sure no group option is set at all (group settings ignored)
		assert.Equal(t, gotGroup, "")
		// make sure the repo flag is used instead
		gotRepo, _ := gotOpts.baseRepo()
		assert.Equal(t, gotRepo.FullName(), "OWNER2/REPO2")
	})
}
