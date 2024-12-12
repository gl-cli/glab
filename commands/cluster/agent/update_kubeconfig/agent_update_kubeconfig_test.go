package update_kubeconfig

import (
	"testing"

	"github.com/stretchr/testify/assert"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

func TestAgent_UpdateKubeConfig_GlabExec(t *testing.T) {
	// GIVEN
	startingConfig := clientcmdapi.NewConfig()

	// WHEN
	params := updateKubeconfigParams{
		startingConfig: *startingConfig,
		glabExecutable: "glab",
		glHost:         "gitlab.example.com",
		glUser:         "gitlab-user",
		kasK8sProxyURL: "https://kas.gitlab.example.com/k8s-proxy",
		agent:          &gitlab.Agent{ID: 42, Name: "test-agent", ConfigProject: gitlab.ConfigProject{PathWithNamespace: "foo/bar"}},
	}
	modifiedConfig, contextName := updateKubeconfig(params)

	// THEN
	// verify the cluster entry
	assert.Contains(t, modifiedConfig.Clusters, "gitlab_example_com")
	assert.Equal(t, "https://kas.gitlab.example.com/k8s-proxy", modifiedConfig.Clusters["gitlab_example_com"].Server)

	// verify auth info entry
	assert.Contains(t, modifiedConfig.AuthInfos, "gitlab_example_com-42")
	actualExec := modifiedConfig.AuthInfos["gitlab_example_com-42"].Exec
	assert.Equal(t, k8sAuthInfoExecApiVersion, actualExec.APIVersion)
	assert.Equal(t, "glab", actualExec.Command)
	assert.Equal(t, []string{"cluster", "agent", "get-token", "--agent", "42"}, actualExec.Args)
	assert.Equal(t, clientcmdapi.NeverExecInteractiveMode, actualExec.InteractiveMode)
	assert.Empty(t, modifiedConfig.AuthInfos["gitlab_example_com-42"].Token)
	assert.Empty(t, modifiedConfig.AuthInfos["gitlab_example_com-42"].TokenFile)

	// verify context entry
	assert.Contains(t, modifiedConfig.Contexts, "gitlab_example_com-foo_bar-test-agent")
	actualContext := modifiedConfig.Contexts["gitlab_example_com-foo_bar-test-agent"]
	assert.Equal(t, "gitlab_example_com", actualContext.Cluster)
	assert.Equal(t, "gitlab_example_com-42", actualContext.AuthInfo)

	// verify returned context name
	assert.Equal(t, "gitlab_example_com-foo_bar-test-agent", contextName)
}
