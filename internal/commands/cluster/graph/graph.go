package graph

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/cli/internal/api"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/internal/iostreams"
	"gitlab.com/gitlab-org/cli/internal/text"
)

var (
	//go:embed long.md
	longHelp string
	//go:embed example.md
	exampleHelp string
)

type options struct {
	io                    *iostreams.IOStreams
	apiClient             func(repoHost string, cfg config.Config) (*api.Client, error)
	baseRepo              func() (glrepo.Interface, error)
	config                func() config.Config
	listenNet, listenAddr string
	agentID               int64
	nsNames               []string
	nsLabels              string
	nsSelector            string
	nsCEL                 string
	resources             []string
	readQueryFromStdIn    bool
	groupCore             bool
	groupBatch            bool
	groupApps             bool
	groupRBAC             bool
	groupClusterRBAC      bool
	groupCRD              bool
	logWatchRequest       bool
}

func NewCmdGraph(f cmdutils.Factory) *cobra.Command {
	opts := options{
		io:        f.IO(),
		apiClient: f.ApiClient,
		baseRepo:  f.BaseRepo,
		config:    f.Config,

		listenNet:  "tcp",
		listenAddr: "localhost:0",
	}
	graphCmd := &cobra.Command{
		Use:     "graph [flags]",
		Short:   `Queries the Kubernetes object graph, using the GitLab Agent for Kubernetes. (EXPERIMENTAL)`,
		Long:    longHelp + text.ExperimentalString,
		Example: exampleHelp,
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return opts.run(cmd.Context())
		},
	}
	fl := graphCmd.Flags()
	fl.Int64VarP(&opts.agentID, "agent", "a", opts.agentID, "The numerical Agent ID to connect to.")
	fl.StringVar(&opts.listenNet, "listen-net", opts.listenNet, "Network on which to listen for connections.")
	fl.StringVar(&opts.listenAddr, "listen-addr", opts.listenAddr, "Address to listen on.")
	fl.BoolVarP(&opts.logWatchRequest, "log-watch-request", "", opts.logWatchRequest, "Log watch request to stdout. Helpful for debugging.")

	fl.StringArrayVarP(&opts.nsNames, "namespace", "n", opts.nsNames, "Namespaces to watch. If not specified, all namespaces are watched with label and field selectors filtering.")
	fl.StringVarP(&opts.nsLabels, "ns-label-selector", "", opts.nsLabels, "Label selector to select namespaces.")
	fl.StringVarP(&opts.nsSelector, "ns-field-selector", "", opts.nsSelector, "Field selector to select namespaces.")
	fl.StringVarP(&opts.nsCEL, "ns-expression", "", opts.nsCEL, "CEL expression to select namespaces. Evaluated before a namespace is watched and on any updates for the namespace object.")

	fl.StringArrayVarP(&opts.resources, "resources", "r", opts.resources, "A list of resources to watch. You can see the list of resources your cluster supports by running 'kubectl api-resources'.")
	fl.BoolVar(&opts.groupCore, "core", opts.groupCore, "Watch pods, secrets, configmaps, and serviceaccounts in the core/v1 group")
	fl.BoolVar(&opts.groupBatch, "batch", opts.groupBatch, "Watch jobs and cronjobs in the batch/v1 group.")
	fl.BoolVar(&opts.groupApps, "apps", opts.groupApps, "Watch deployments, replicasets, daemonsets, and statefulsets in apps/v1 group.")
	fl.BoolVar(&opts.groupRBAC, "rbac", opts.groupRBAC, "Watch roles, and rolebindings in the rbac.authorization.k8s.io/v1 group.")
	fl.BoolVar(&opts.groupClusterRBAC, "cluster-rbac", opts.groupClusterRBAC, "Watch clusterroles and clusterrolebindings in the rbac.authorization.k8s.io/v1 group.")
	fl.BoolVar(&opts.groupCRD, "crd", opts.groupCRD, "Watch customresourcedefinitions in the apiextensions.k8s.io/v1 group.")
	fl.BoolVar(&opts.readQueryFromStdIn, "stdin", opts.readQueryFromStdIn, "Read watch request from standard input.")

	cobra.CheckErr(graphCmd.MarkFlagRequired("agent"))
	graphCmd.MarkFlagsMutuallyExclusive("stdin", "namespace")
	graphCmd.MarkFlagsMutuallyExclusive("stdin", "ns-label-selector")
	graphCmd.MarkFlagsMutuallyExclusive("stdin", "ns-field-selector")
	graphCmd.MarkFlagsMutuallyExclusive("stdin", "ns-expression")
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
	cfg := o.config()
	client, err := o.apiClient(repo.RepoHost(), cfg)
	if err != nil {
		return err
	}

	// 2. Check token type
	authSource, ok := client.AuthSource().(gitlab.AccessTokenAuthSource)
	if !ok {
		return errors.New("cluster graph command supports authentication with only personal and project access tokens. Requires at least the Developer role.")
	}

	// 3. Read the watch request
	watchReq, err := o.constructWatchRequest()
	if err != nil {
		return err
	}

	if o.logWatchRequest {
		o.io.LogInfo(string(watchReq))
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
		authorization: fmt.Sprintf("Bearer pat:%d:%s", o.agentID, authSource.Token),
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
		Queries:    q,
		Namespaces: o.constructWatchNamespaces(),
	})
	if err != nil {
		return nil, fmt.Errorf("JSON marshal: %w", err)
	}
	return req, nil
}

func (o *options) constructWatchNamespaces() *namespaces {
	if o.isNamespaceOptsEmpty() {
		return &namespaces{
			ObjectSelectorExpression: "name != 'kube-system'",
		}
	}
	return &namespaces{
		Names:                    o.nsNames,
		LabelSelector:            o.nsLabels,
		FieldSelector:            o.nsSelector,
		ObjectSelectorExpression: o.nsCEL,
	}
}

func (o *options) isNamespaceOptsEmpty() bool {
	return len(o.nsNames) == 0 && o.nsLabels == "" && o.nsSelector == "" && o.nsCEL == ""
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
