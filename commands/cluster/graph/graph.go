package graph

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/pkg/iostreams"
	"gitlab.com/gitlab-org/cli/pkg/text"
)

type options struct {
	io                    *iostreams.IOStreams
	baseRepo              func() (glrepo.Interface, error)
	config                func() (config.Config, error)
	listenNet, listenAddr string
	agentID               int64
	resources             []string
	readQueryFromStdIn    bool
	groupCore             bool
	groupBatch            bool
	groupApps             bool
	groupRBAC             bool
	groupClusterRBAC      bool
	groupCRD              bool
}

func NewCmdGraph(f cmdutils.Factory) *cobra.Command {
	opts := options{
		listenNet:  "tcp",
		listenAddr: "localhost:0",
	}
	graphCmd := &cobra.Command{
		Use:   "graph [flags]",
		Short: `Query Kubernetes object graph using GitLab Agent for Kubernetes. (EXPERIMENTAL)`,
		Long: heredoc.Doc(`
This commands starts a web server that shows a live view of Kubernetes objects graph in a browser.
It works via the GitLab Agent for Kubernetes running in the cluster.
The minimum required GitLab and GitLab Agent version is v18.1.

This command only supports personal and project access tokens for authentication.
` + "The token should have at least the `Developer` role and the `read_api` and `k8s_proxy` scopes.\n" +
			text.ExperimentalString),
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.io = f.IO() // TODO move into the struct literal after factory refactoring
			opts.baseRepo = f.BaseRepo
			opts.config = f.Config
			return opts.run(cmd.Context())
		},
	}
	fl := graphCmd.Flags()
	fl.Int64VarP(&opts.agentID, "agent", "a", opts.agentID, "The numerical Agent ID to connect to.")
	fl.StringVar(&opts.listenNet, "listen-net", opts.listenNet, "Network on which to listen for connections.")
	fl.StringVar(&opts.listenAddr, "listen-addr", opts.listenAddr, "Address to listen on.")

	fl.StringArrayVarP(&opts.resources, "resources", "r", opts.resources, "A list of resources to watch. You can see the list of resources your cluster supports by running kubectl api-resources.")
	fl.BoolVar(&opts.groupCore, "core", opts.groupCore, "Watch pods, secrets, configmaps, and serviceaccounts in core/v1 group")
	fl.BoolVar(&opts.groupBatch, "batch", opts.groupBatch, "Watch jobs, and cronjobs in batch/v1 group.")
	fl.BoolVar(&opts.groupApps, "apps", opts.groupApps, "Watch deployments, replicasets, daemonsets, and statefulsets in apps/v1 group.")
	fl.BoolVar(&opts.groupRBAC, "rbac", opts.groupRBAC, "Watch roles, and rolebindings in rbac.authorization.k8s.io/v1 group.")
	fl.BoolVar(&opts.groupClusterRBAC, "cluster-rbac", opts.groupClusterRBAC, "Watch clusterroles, and clusterrolebindings in rbac.authorization.k8s.io/v1 group.")
	fl.BoolVar(&opts.groupCRD, "crd", opts.groupCRD, "Watch customresourcedefinitions in apiextensions.k8s.io/v1 group.")
	fl.BoolVar(&opts.readQueryFromStdIn, "stdin", opts.readQueryFromStdIn, "Read watch request from standard input.")

	cobra.CheckErr(graphCmd.MarkFlagRequired("agent"))
	graphCmd.MarkFlagsMutuallyExclusive("stdin", "resources")
	graphCmd.MarkFlagsMutuallyExclusive("stdin", "core")
	graphCmd.MarkFlagsMutuallyExclusive("stdin", "batch")
	graphCmd.MarkFlagsMutuallyExclusive("stdin", "apps")
	graphCmd.MarkFlagsMutuallyExclusive("stdin", "rbac")
	graphCmd.MarkFlagsMutuallyExclusive("stdin", "cluster-rbac")
	graphCmd.MarkFlagsMutuallyExclusive("stdin", "crd")

	return graphCmd
}

func (o *options) run(ctx context.Context) error {
	// 1. Plumbing setup
	repo, err := o.baseRepo()
	if err != nil {
		return err
	}
	cfg, err := o.config()
	if err != nil {
		return err
	}
	client, err := api.NewClientWithCfg(repo.RepoHost(), cfg, false)
	if err != nil {
		return err
	}

	// 2. Check token type
	if client.AuthType != api.PrivateToken {
		return errors.New("cluster graph command supports authentication with personal and project access tokens only (with Developer+ role)")
	}

	// 3. Read the watch request
	watchReq, err := o.constructWatchRequest()
	if err != nil {
		return err
	}

	// 4. Construct API URL
	md, _, err := client.Lab().Metadata.GetMetadata(gitlab.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("GitLab metadata: %w", err)
	}
	graphAPIURL := md.KAS.ExternalK8SProxyURL
	if !strings.HasSuffix(graphAPIURL, "/") {
		graphAPIURL += "/"
	}
	graphAPIURL += "graph"

	// 5. Start the server
	srv := server{
		log:           slog.New(slog.NewTextHandler(o.io.StdErr, nil)),
		io:            o.io,
		httpClient:    client.HTTPClient(),
		graphAPIURL:   graphAPIURL,
		listenNet:     o.listenNet,
		listenAddr:    o.listenAddr,
		authorization: fmt.Sprintf("Bearer pat:%d:%s", o.agentID, client.Token()),
		watchRequest:  watchReq,
	}
	return srv.Run(ctx)
}

func (o *options) constructWatchRequest() ([]byte, error) {
	if o.readQueryFromStdIn {
		return o.readWatchRequestFromStdin()
	}

	q := o.maybeConstructWatchQueriesForGroups()
	q = append(q, o.maybeConstructWatchQueriesForResources()...)
	if len(q) == 0 {
		q = o.defaultWatchQueries()
	}

	req, err := json.Marshal(&watchGraphWebSocketRequest{
		Queries: q,
		Namespaces: &namespaces{
			ObjectSelectorExpression: "name != 'kube-system'",
		},
	})
	if err != nil {
		return nil, fmt.Errorf("JSON marshal: %w", err)
	}
	return req, nil
}

func (o *options) readWatchRequestFromStdin() ([]byte, error) {
	req, err := io.ReadAll(o.io.In)
	if err != nil {
		return nil, fmt.Errorf("reading request from stdin: %w", err)
	}
	return req, nil
}

func (o *options) defaultWatchQueries() []query {
	return []query{
		{
			Include: &queryInclude{
				ResourceSelectorExpression: "group == '' && version == 'v1' && (resource in ['pods', 'secrets', 'configmaps', 'serviceaccounts'])",
			},
		},
		{
			Include: &queryInclude{
				ResourceSelectorExpression: "group == 'apps' && version == 'v1' && (resource in ['deployments', 'replicasets', 'daemonsets', 'statefulsets'])",
			},
		},
		{
			Include: &queryInclude{
				ResourceSelectorExpression: "group == 'batch' && version == 'v1' && (resource in ['jobs', 'cronjobs'])",
			},
		},
		{
			Include: &queryInclude{
				ResourceSelectorExpression: "group == 'rbac.authorization.k8s.io' && version == 'v1' && !(resource in ['clusterrolebindings', 'clusterroles'])",
			},
		},
	}
}

func (o *options) maybeConstructWatchQueriesForResources() []query {
	if len(o.resources) == 0 {
		return nil
	}
	var sb strings.Builder
	sb.WriteString("resource in [")
	for i, resource := range o.resources {
		if i == 0 {
			sb.WriteByte('\'')
		} else {
			sb.WriteString(",'")
		}
		sb.WriteString(resource)
		sb.WriteByte('\'')
	}
	sb.WriteByte(']')

	return []query{
		{
			Include: &queryInclude{
				ResourceSelectorExpression: sb.String(),
			},
		},
	}
}

func (o *options) maybeConstructWatchQueriesForGroups() []query {
	var q []query

	if o.groupCore {
		q = append(q, query{
			Include: &queryInclude{
				ResourceSelectorExpression: "group == '' && version == 'v1' && (resource in ['pods', 'secrets', 'configmaps', 'serviceaccounts'])",
			},
		})
	}
	if o.groupBatch {
		q = append(q, query{
			Include: &queryInclude{
				ResourceSelectorExpression: "group == 'batch' && version == 'v1' && (resource in ['jobs', 'cronjobs'])",
			},
		})
	}
	if o.groupApps {
		q = append(q, query{
			Include: &queryInclude{
				ResourceSelectorExpression: "group == 'apps' && version == 'v1' && (resource in ['deployments', 'replicasets', 'daemonsets', 'statefulsets'])",
			},
		})
	}
	if o.groupRBAC {
		q = append(q, query{
			Include: &queryInclude{
				ResourceSelectorExpression: "group == 'rbac.authorization.k8s.io' && version == 'v1' && (resource in ['roles', 'rolebindings'])",
			},
		})
	}
	if o.groupClusterRBAC {
		q = append(q, query{
			Include: &queryInclude{
				ResourceSelectorExpression: "group == 'rbac.authorization.k8s.io' && version == 'v1' && (resource in ['clusterroles', 'clusterrolebindings'])",
			},
		})
	}
	if o.groupCRD {
		q = append(q, query{
			Include: &queryInclude{
				ResourceSelectorExpression: "group == 'apiextensions.k8s.io' && version == 'v1' && resource == 'customresourcedefinitions'",
			},
		})
	}
	return q
}
