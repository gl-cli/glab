package revoke

import (
	"context"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/internal/iostreams"
)

type options struct {
	io           *iostreams.IOStreams
	gitlabClient func() (*gitlab.Client, error)
	baseRepoFunc func() (glrepo.Interface, error)

	agentID int64
	tokenID int64
}

func NewCmd(f cmdutils.Factory) *cobra.Command {
	opts := options{
		io:           f.IO(),
		gitlabClient: f.GitLabClient,
		baseRepoFunc: f.BaseRepo,
	}

	cmd := &cobra.Command{
		Use:   "revoke <agent-id> <token-id>",
		Short: `Revoke a token of an agent.`,
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := opts.complete(args); err != nil {
				return err
			}

			return opts.run(cmd.Context())
		},
	}

	return cmd
}

func (o *options) complete(args []string) error {
	agentID, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("agent ID must be a valid integer, got %q", args[0])
	}

	o.agentID = agentID

	tokenID, err := strconv.ParseInt(args[1], 10, 64)
	if err != nil {
		return fmt.Errorf("token ID must be a valid integer, got %q", args[1])
	}

	o.tokenID = tokenID

	return nil
}

func (o *options) run(ctx context.Context) error {
	client, err := o.gitlabClient()
	if err != nil {
		return err
	}

	baseRepo, err := o.baseRepoFunc()
	if err != nil {
		return err
	}

	_, err = client.ClusterAgents.RevokeAgentToken(
		baseRepo.FullName(),
		int(o.agentID),
		int(o.tokenID),
		gitlab.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("failed to revoke token: %w", err)
	}

	fmt.Fprintf(
		o.io.StdOut,
		"Successfully revoked token %d of agent %d\n",
		o.tokenID,
		o.agentID,
	)

	return nil
}
