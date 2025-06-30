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
	"gitlab.com/gitlab-org/cli/internal/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/pkg/iostreams"
	"gitlab.com/gitlab-org/cli/pkg/text"
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
	objSelExprHelp := strings.Join([]string{
		"- `obj` is the Kubernetes object being evaluated.",
		"- `group` group of the object.",
		"- `version` version of the object.",
		"- `resource` resource name of the object. E.g. pods for the Pod kind.",
		"- `namespace` namespace of the object.",
		"- `name` name of the object.",
		"- `labels` labels of the object.",
		"- `annotations` annotations of the object.",
	}, "\n")
	resSelExprHelp := strings.Join([]string{
		"- `group` group of the object.",
		"- `version` version of the object.",
		"- `resource` resource name of the object. E.g. pods for the Pod kind.",
		"- `namespaced` scope of group+version+resource. Can be `bool` `true` or `false`.",
	}, "\n")
	graphCmd := &cobra.Command{
		Use:   "graph [flags]",
		Short: `Query Kubernetes object graph using GitLab Agent for Kubernetes. (EXPERIMENTAL)`,
		Long: heredoc.Docf(`
		This commands starts a web server that shows a live view of Kubernetes objects graph in a browser.
		It works via the GitLab Agent for Kubernetes running in the cluster.
		The minimum required GitLab and GitLab Agent version is v18.1.

		Please leave feedback in [this issue](https://gitlab.com/gitlab-org/cli/-/issues/7900).

		### Resource filtering

		Resources and namespaces can be filterer using [CEL expressions](https://cel.dev/).

		%s can be used to filter objects. The expression must return a boolean. The following variables are available:

		%s

		%s can be used to filter Kubernetes discovery information to include/exclude resources
		from the watch request. The expression must return a boolean. The following variables are available:

		%s

		### Advanced usage

		Apart from high level ways to construct the query, this command allows you to construct and send
		the query using all the underlying API capabilities.
		Please see the
		[technical design doc](https://gitlab.com/gitlab-org/cluster-integration/gitlab-agent/-/blob/master/doc/graph_api.md)
		to understand what is possible and how to do it.

		This command only supports personal and project access tokens for authentication.
		`+"The token should have at least the `Developer` role in the agent project and the `read_api` and `k8s_proxy` scopes."+`
		The user should be allowed to access the agent project.
		See <https://docs.gitlab.com/user/clusters/agent/user_access/>. 
		%s`, "`object_selector_expression`", objSelExprHelp, "`resource_selector_expression`", resSelExprHelp, text.ExperimentalString),
		Example: heredoc.Doc(`
		# Run the default query for agent 123
		$ glab cluster graph -R user/project -a 123

		# Show common resources from the core and RBAC groups
		$ glab cluster graph -R user/project -a 123 --core --rbac

		# Show certain resources
		$ glab cluster graph -R user/project -a 123 --resources=pods --resources=configmaps

		# Same as above, but more compact
		$ glab cluster graph -R user/project -a 123 -r={pods,configmaps}

		# Select a certain namespace
		$ glab cluster graph -R user/project -a 123 -n={my-ns,my-stuff}

		# Select all namespaces that have a certain annotation
		$ glab cluster graph -R user/project -a 123 --ns-expression='"my-annotation" in annotations'

		# Advanced usage - pass the full query directly via stdin.
		# The query below watches serviceaccounts in all namespaces except for the kube-system.
		$ Q='{"queries":[{"include":{"resource_selector_expression":"resource == \"serviceaccounts\""}}],"namespaces":{"object_selector_expression":"name != \"kube-system\""}}'

		$ echo -n "$Q" | glab cluster graph -R user/project -a 123 --stdin
		`),
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return opts.run(cmd.Context())
		},
	}
	fl := graphCmd.Flags()
	fl.Int64VarP(&opts.agentID, "agent", "a", opts.agentID, "The numerical Agent ID to connect to.")
	fl.StringVar(&opts.listenNet, "listen-net", opts.listenNet, "Network on which to listen for connections.")
	fl.StringVar(&opts.listenAddr, "listen-addr", opts.listenAddr, "Address to listen on.")
	fl.BoolVarP(&opts.logWatchRequest, "log-watch-request", "", opts.logWatchRequest, "Log watch request to stdout. Can be useful for debugging.")

	fl.StringArrayVarP(&opts.nsNames, "namespace", "n", opts.nsNames, "Namespaces to watch. If not specified, all namespaces are watched with label and field selectors filtering.")
	fl.StringVarP(&opts.nsLabels, "ns-label-selector", "", opts.nsLabels, "Label selector to select namespaces. See https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#label-selectors.")
	fl.StringVarP(&opts.nsSelector, "ns-field-selector", "", opts.nsSelector, "Field selector to select namespaces. See https://kubernetes.io/docs/concepts/overview/working-with-objects/field-selectors/.")
	fl.StringVarP(&opts.nsCEL, "ns-expression", "", opts.nsCEL, "CEL expression to select namespaces. Evaluated before a namespace is watched and on any updates for the namespace object.")

	fl.StringArrayVarP(&opts.resources, "resources", "r", opts.resources, "A list of resources to watch. You can see the list of resources your cluster supports by running kubectl api-resources.")
	fl.BoolVar(&opts.groupCore, "core", opts.groupCore, "Watch pods, secrets, configmaps, and serviceaccounts in core/v1 group")
	fl.BoolVar(&opts.groupBatch, "batch", opts.groupBatch, "Watch jobs, and cronjobs in batch/v1 group.")
	fl.BoolVar(&opts.groupApps, "apps", opts.groupApps, "Watch deployments, replicasets, daemonsets, and statefulsets in apps/v1 group.")
	fl.BoolVar(&opts.groupRBAC, "rbac", opts.groupRBAC, "Watch roles, and rolebindings in rbac.authorization.k8s.io/v1 group.")
	fl.BoolVar(&opts.groupClusterRBAC, "cluster-rbac", opts.groupClusterRBAC, "Watch clusterroles, and clusterrolebindings in rbac.authorization.k8s.io/v1 group.")
	fl.BoolVar(&opts.groupCRD, "crd", opts.groupCRD, "Watch customresourcedefinitions in apiextensions.k8s.io/v1 group.")
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
	if client.AuthType != api.PrivateToken {
		return errors.New("cluster graph command supports authentication with personal and project access tokens only (with Developer+ role)")
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
