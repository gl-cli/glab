package update_kubeconfig

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/commands/cluster/agent/agentutils"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/internal/iostreams"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

const (
	k8sAuthInfoExecApiVersion = "client.authentication.k8s.io/v1"

	flagAgent      = "agent"
	flagUseContext = "use-context"
)

var sanitizeReplacer = strings.NewReplacer("/", "_", ".", "_")

type options struct {
	httpClient   func() (*gitlab.Client, error)
	baseRepo     func() (glrepo.Interface, error)
	io           *iostreams.IOStreams
	configAccess clientcmd.ConfigAccess

	useContext bool
	agentID    int64
}

func NewCmdAgentUpdateKubeconfig(f cmdutils.Factory) *cobra.Command {
	pathOptions := clientcmd.NewDefaultPathOptions()
	if len(pathOptions.ExplicitFileFlag) == 0 {
		pathOptions.ExplicitFileFlag = clientcmd.RecommendedConfigPathFlag
	}

	opts := options{
		io:           f.IO(),
		httpClient:   f.HttpClient,
		baseRepo:     f.BaseRepo,
		configAccess: pathOptions,
	}

	agentUpdateKubeconfigCmd := &cobra.Command{
		Use:   "update-kubeconfig [flags]",
		Short: `Update selected kubeconfig.`,
		Long: heredoc.Doc(`Update selected kubeconfig for use with a GitLab agent for Kubernetes.
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			return opts.run()
		},
	}
	agentUpdateKubeconfigCmd.Flags().Int64VarP(&opts.agentID, flagAgent, "a", opts.agentID, "The numeric agent ID to create the kubeconfig entry for.")
	cobra.CheckErr(agentUpdateKubeconfigCmd.MarkFlagRequired(flagAgent))
	persistentFlags := agentUpdateKubeconfigCmd.PersistentFlags()
	persistentFlags.StringVar(&pathOptions.LoadingRules.ExplicitPath, pathOptions.ExplicitFileFlag, pathOptions.LoadingRules.ExplicitPath, "Use a particular kubeconfig file.")
	persistentFlags.BoolVarP(&opts.useContext, flagUseContext, "u", opts.useContext, "Use as default context.")

	return agentUpdateKubeconfigCmd
}

func (o *options) run() error {
	apiClient, err := o.httpClient()
	if err != nil {
		return err
	}

	repo, err := o.baseRepo()
	if err != nil {
		return err
	}

	// Retrieve metadata of the instance to determine KAS URL
	metadata, _, err := apiClient.Metadata.GetMetadata()
	if err != nil {
		return err
	}
	if !metadata.KAS.Enabled {
		return fmt.Errorf("the GitLab agent server for Kubernetes is disabled on %s. Ask your administrator to enable and configure it.", repo.RepoHost())
	}
	kasK8SProxyURL, err := agentutils.GetKasK8SProxyURL(metadata)
	if err != nil {
		return err
	}

	// Retrieve agent information, most importantly its name to use it as context name.
	repoFullName := repo.FullName()
	agent, _, err := apiClient.ClusterAgents.GetAgent(repoFullName, int(o.agentID)) // FIXME remove cast
	if err != nil {
		return err
	}

	// Retrieve user information
	user, _, err := apiClient.Users.CurrentUser()
	if err != nil {
		return err
	}

	// Retrieve glab executable path for exec
	glabExecutable, err := os.Executable()
	if err != nil {
		return nil
	}

	startingConfig, err := o.configAccess.GetStartingConfig()
	if err != nil {
		return err
	}

	params := updateKubeconfigParams{
		startingConfig: *startingConfig,
		glabExecutable: glabExecutable,
		glHost:         repo.RepoHost(),
		glRepoFullName: repoFullName,
		glUser:         user.Username,
		kasK8sProxyURL: kasK8SProxyURL,
		agent:          agent,
	}
	config, contextName := updateKubeconfig(params)

	if o.useContext {
		config.CurrentContext = contextName
	}

	if err := clientcmd.ModifyConfig(o.configAccess, config, true); err != nil {
		return err
	}

	o.io.LogInfof("Updated context %s.\n", contextName)

	if o.useContext {
		o.io.LogInfof("Using context %s.\n", contextName)
	}
	return nil
}

type updateKubeconfigParams struct {
	startingConfig clientcmdapi.Config
	glabExecutable string
	glHost         string
	glRepoFullName string
	glUser         string
	kasK8sProxyURL string
	agent          *gitlab.Agent
}

func updateKubeconfig(params updateKubeconfigParams) (clientcmdapi.Config, string) {
	config := params.startingConfig

	// Updating `clusters` entry: `kubectl config set-cluster ...`
	clusterName := sanitizeForKubeconfig(params.glHost)
	startingCluster, exists := config.Clusters[clusterName]
	if !exists {
		startingCluster = clientcmdapi.NewCluster()
	}
	config.Clusters[clusterName] = modifyCluster(*startingCluster, params.kasK8sProxyURL)

	// Updating `users` entry: `kubectl config set-credentials ...`
	authInfoName := fmt.Sprintf("%s-%d", clusterName, params.agent.ID)
	startingAuthInfo, exists := config.AuthInfos[authInfoName]
	if !exists {
		startingAuthInfo = clientcmdapi.NewAuthInfo()
	}
	config.AuthInfos[authInfoName] = modifyAuthInfo(*startingAuthInfo, params.glabExecutable, params.glHost, params.glRepoFullName, int64(params.agent.ID)) // FIXME remove cast

	// Updating `contexts` entry: `kubectl config set-context ...`
	contextName := fmt.Sprintf("%s-%s-%s", clusterName, sanitizeForKubeconfig(params.agent.ConfigProject.PathWithNamespace), params.agent.Name)
	startingContext, exists := config.Contexts[contextName]
	if !exists {
		startingContext = clientcmdapi.NewContext()
	}
	config.Contexts[contextName] = modifyContext(*startingContext, clusterName, authInfoName)

	return config, contextName
}

func modifyCluster(cluster clientcmdapi.Cluster, server string) *clientcmdapi.Cluster {
	cluster.Server = server
	return &cluster
}

func modifyAuthInfo(authInfo clientcmdapi.AuthInfo, glabExecutable string, glabHost string, glRepoFullName string, agentID int64) *clientcmdapi.AuthInfo {
	// Clear existing auth info
	authInfo.Token = ""
	authInfo.TokenFile = ""

	// Two reasons to set --repo and GITLAB_HOST:
	// - Propagate the host and repo if it's not the default one.
	// - Isolate from the variable(s) that might be set to a different value when kubectl calls this command to get a token.
	//   This requires ALWAYS setting this variable, even it has the default value as the caller might have
	//   a custom value set, but it must be ignored.

	authInfo.Exec = &clientcmdapi.ExecConfig{
		Command: glabExecutable,
		Args: []string{
			"cluster", "agent", "get-token",
			"--agent", strconv.FormatInt(agentID, 10),
			"--repo", glRepoFullName,
		},
		Env: []clientcmdapi.ExecEnvVar{
			{
				Name:  "GITLAB_HOST",
				Value: glabHost,
			},
		},
		APIVersion: k8sAuthInfoExecApiVersion,
		InstallHint: heredoc.Doc(`
			To authenticate to the current cluster, glab is required.

			Follow the installation instructions at https://gitlab.com/gitlab-org/cli#installation.
		`),
		InteractiveMode: clientcmdapi.NeverExecInteractiveMode,
	}

	return &authInfo
}

func modifyContext(ctx clientcmdapi.Context, clusterName, authInfoName string) *clientcmdapi.Context {
	ctx.Cluster = clusterName
	ctx.AuthInfo = authInfoName
	return &ctx
}

func sanitizeForKubeconfig(name string) string {
	return sanitizeReplacer.Replace(name)
}
