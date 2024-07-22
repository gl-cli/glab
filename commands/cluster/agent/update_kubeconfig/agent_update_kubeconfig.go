package update_kubeconfig

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/MakeNowJust/heredoc/v2"

	"github.com/spf13/cobra"
	"github.com/xanzy/go-gitlab"
	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

const (
	kasProxyProtocol = "https"
	kasProxyEndpoint = "k8s-proxy"

	k8sAuthInfoExecApiVersion = "client.authentication.k8s.io/v1"

	flagAgent      = "agent"
	flagUseContext = "use-context"
)

var sanitizeReplacer = strings.NewReplacer("/", "_", ".", "_")

func NewCmdAgentUpdateKubeconfig(f *cmdutils.Factory) *cobra.Command {
	pathOptions := clientcmd.NewDefaultPathOptions()
	if len(pathOptions.ExplicitFileFlag) == 0 {
		pathOptions.ExplicitFileFlag = clientcmd.RecommendedConfigPathFlag
	}

	agentUpdateKubeconfigCmd := &cobra.Command{
		Use:   "update-kubeconfig [flags]",
		Short: `Update selected kubeconfig.`,
		Long: heredoc.Doc(`Update selected kubeconfig for use with a GitLab agent for Kubernetes.
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			agentID, err := cmd.Flags().GetInt(flagAgent)
			if err != nil {
				return err
			}

			useContext, err := cmd.Flags().GetBool(flagUseContext)
			if err != nil {
				return err
			}

			return runUpdateKubeconfig(agentID, pathOptions, useContext, f)
		},
	}
	agentUpdateKubeconfigCmd.Flags().IntP(flagAgent, "a", 0, "The numeric agent ID to create the kubeconfig entry for.")
	cobra.CheckErr(agentUpdateKubeconfigCmd.MarkFlagRequired(flagAgent))
	persistentFlags := agentUpdateKubeconfigCmd.PersistentFlags()
	persistentFlags.StringVar(&pathOptions.LoadingRules.ExplicitPath, pathOptions.ExplicitFileFlag, pathOptions.LoadingRules.ExplicitPath, "Use a particular kubeconfig file.")
	persistentFlags.BoolP(flagUseContext, "u", false, "Use as default context.")

	return agentUpdateKubeconfigCmd
}

func runUpdateKubeconfig(agentID int, configAccess clientcmd.ConfigAccess, useContext bool, factory *cmdutils.Factory) error {
	apiClient, err := factory.HttpClient()
	if err != nil {
		return err
	}

	repo, err := factory.BaseRepo()
	if err != nil {
		return err
	}

	// Retrieve metadata of the instance to determine KAS URL
	metadata, err := api.GetMetadata(apiClient)
	if err != nil {
		return err
	}
	if !metadata.KAS.Enabled {
		return fmt.Errorf("the GitLab agent server for Kubernetes is disabled on %s. Ask your administrator to enable and configure it.", repo.RepoHost())
	}
	kasUrl, err := url.Parse(metadata.KAS.ExternalURL)
	if err != nil {
		return err
	}

	// Retrieve agent information, most importantly its name to use it as context name.
	agent, err := api.GetAgent(apiClient, repo.FullName(), agentID)
	if err != nil {
		return err
	}

	// Retrieve user information
	user, err := api.CurrentUser(apiClient)
	if err != nil {
		return err
	}

	// Retrieve glab executable path for exec
	glabExecutable, err := os.Executable()
	if err != nil {
		return nil
	}

	startingConfig, err := configAccess.GetStartingConfig()
	if err != nil {
		return err
	}

	params := updateKubeconfigParams{
		startingConfig: *startingConfig,
		glabExecutable: glabExecutable,
		glHost:         repo.RepoHost(),
		glKasUrl:       kasUrl,
		glUser:         user.Username,
		agent:          agent,
	}
	config, contextName := updateKubeconfig(params)

	if useContext {
		config.CurrentContext = contextName
	}

	if err := clientcmd.ModifyConfig(configAccess, config, true); err != nil {
		return err
	}

	fmt.Fprintf(factory.IO.StdOut, "Updated context %s.\n", contextName)

	if useContext {
		fmt.Fprintf(factory.IO.StdOut, "Using context %s.\n", contextName)
	}
	return nil
}

type updateKubeconfigParams struct {
	startingConfig clientcmdapi.Config
	glabExecutable string
	glHost         string
	glKasUrl       *url.URL
	glUser         string
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
	config.Clusters[clusterName] = modifyCluster(*startingCluster, constructKasProxyURL(params.glKasUrl))

	// Updating `users` entry: `kubectl config set-credentials ...`
	authInfoName := fmt.Sprintf("%s-%d", clusterName, params.agent.ID)
	startingAuthInfo, exists := config.AuthInfos[authInfoName]
	if !exists {
		startingAuthInfo = clientcmdapi.NewAuthInfo()
	}
	config.AuthInfos[authInfoName] = modifyAuthInfo(*startingAuthInfo, params.glabExecutable, params.agent.ID)

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

func modifyAuthInfo(authInfo clientcmdapi.AuthInfo, glabExecutable string, agentID int) *clientcmdapi.AuthInfo {
	// Clear existing auth info
	authInfo.Token = ""
	authInfo.TokenFile = ""

	authInfo.Exec = &clientcmdapi.ExecConfig{
		APIVersion:      k8sAuthInfoExecApiVersion,
		Command:         glabExecutable,
		Args:            []string{"cluster", "agent", "get-token", "--agent", strconv.Itoa(agentID)},
		InteractiveMode: clientcmdapi.NeverExecInteractiveMode,
		InstallHint: heredoc.Doc(`
			To authenticate to the current cluster, glab is required.

			Follow the installation instructions at https://gitlab.com/gitlab-org/cli#installation.
		`),
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

func constructKasProxyURL(u *url.URL) string {
	ku := *u.JoinPath(kasProxyEndpoint)
	ku.Scheme = kasProxyProtocol
	return ku.String()
}
