package bootstrap

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"

	glab_api "gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"

	"github.com/spf13/cobra"
	"github.com/xanzy/go-gitlab"
)

//go:generate go run go.uber.org/mock/mockgen@v0.4.0 -typed -destination=./mocks_for_test.go -package=bootstrap gitlab.com/gitlab-org/cli/commands/cluster/agent/bootstrap API,FluxWrapper,KubectlWrapper,Cmd
//go:generate go run go.uber.org/mock/mockgen@v0.4.0 -typed -destination=./stdlib_mocks_for_test.go -package=bootstrap "io" "Writer"

type API interface {
	GetDefaultBranch() (string, error)
	GetAgentByName(name string) (*gitlab.Agent, error)
	RegisterAgent(name string) (*gitlab.Agent, error)
	ConfigureAgent(agent *gitlab.Agent, branch string) error
	ConfigureEnvironment(agentID int, name string, kubernetesNamespace string, fluxResourcePath string) error
	CreateAgentToken(agentID int) (*gitlab.AgentToken, error)
	SyncFile(f file, branch string) error
	GetKASAddress() (string, error)
}

type FluxWrapper interface {
	createHelmRepositoryManifest() (file, error)
	createHelmReleaseManifest(kasAddress string) (file, error)
	reconcile() error
}

type KubectlWrapper interface {
	createAgentTokenSecret(tokenID int, token string) error
}

type (
	APIFactory            func(*gitlab.Client, any) API
	KubectlWrapperFactory func(Cmd, string, string, string) KubectlWrapper
	FluxWrapperFactory    func(Cmd, string, string, string, string, string, string, string, string, string, []string, []string, string, string, string) FluxWrapper
	CmdFactory            func(io.Writer, io.Writer, []string) Cmd
)

var reconcileErr = errors.New("failed to reconcile the GitLab Agent")

const (
	kubectlBinaryName = "kubectl"
	fluxBinaryName    = "flux"
)

func EnsureRequirements() error {
	if _, err := exec.LookPath(kubectlBinaryName); err != nil {
		return fmt.Errorf("unable to find %s binary in PATH", kubectlBinaryName)
	}
	if _, err := exec.LookPath(fluxBinaryName); err != nil {
		return fmt.Errorf("unable to find %s binary in PATH", fluxBinaryName)
	}
	return nil
}

func NewCmdAgentBootstrap(f *cmdutils.Factory, ensureRequirements func() error, af APIFactory, kwf KubectlWrapperFactory, fwf FluxWrapperFactory, cf CmdFactory) *cobra.Command {
	agentBootstrapCmd := &cobra.Command{
		Use:   "bootstrap agent-name [flags]",
		Short: `Bootstrap a GitLab Agent for Kubernetes in a project.`,
		Long: `Bootstrap a GitLab Agent for Kubernetes (agentk) in a project.

The first argument must be the name of the agent.

It requires the kubectl and flux commands to be accessible via $PATH.

This command consists of multiple idempotent steps:

1. Register the agent with the project.
1. Configure the agent.
1. Configure an environment with dashboard for the agent.
1. Create a token for the agent.
   - If the agent has reached the maximum amount of tokens,
     the one that has not been used the longest is revoked
     and a new one is created.
   - If the agent has not reached the maximum amount of tokens,
     a new one is created.
1. Push the Kubernetes Secret that contains the token to the cluster.
1. Create Flux HelmRepository and HelmRelease resource.
1. Commit and Push the created Flux Helm resources to the manifest path.
1. Trigger Flux reconciliation of GitLab Agent HelmRelease.
`,
		Example: `
# Bootstrap "my-agent" to root of Git project in CWD and trigger reconciliation
glab cluster agent bootstrap my-agent

# Bootstrap "my-agent" to "manifests/" of Git project in CWD and trigger reconciliation
glab cluster agent bootstrap my-agent --manifest-path manifests/

# Bootstrap "my-agent" to "manifests/" of Git project in CWD and do not manually trigger a reconilication
glab cluster agent bootstrap my-agent --manifest-path manifests/ --no-reconcile

# Bootstrap "my-agent" without configuring an environment
glab cluster agent bootstrap my-agent --create-environment=false

# Bootstrap "my-agent" and configure an environment with custom name and Kubernetes namespace
glab cluster agent bootstrap my-agent --environment-name production --environment-namespace default

# Bootstrap "my-agent" and pass additional GitLab Helm Chart values from a local file
glab cluster agent bootstrap my-agent --helm-release-values values.yaml

# Bootstrap "my-agent" and pass additional GitLab Helm Chart values from a Kubernetes ConfigMap
glab cluster agent bootstrap my-agent --helm-release-values-from ConfigMap/agent-config
`,
		Aliases: []string{"bs"},
		Args:    cobra.ExactArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return ensureRequirements()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			stdout, stderr := f.IO.StdOut, f.IO.StdErr

			apiClient, err := f.HttpClient()
			if err != nil {
				return err
			}

			repo, err := f.BaseRepo()
			if err != nil {
				return err
			}

			api := af(apiClient, repo.FullName())

			manifestPath, err := cmd.Flags().GetString("manifest-path")
			if err != nil {
				return err
			}
			manifestBranch, err := cmd.Flags().GetString("manifest-branch")
			if err != nil {
				return err
			}

			if manifestBranch == "" {
				manifestBranch, err = api.GetDefaultBranch()
				if err != nil {
					return err
				}
			}

			noReconcile, err := cmd.Flags().GetBool("no-reconcile")
			if err != nil {
				return err
			}

			helmRepositoryName, err := cmd.Flags().GetString("helm-repository-name")
			if err != nil {
				return err
			}
			helmRepositoryNamespace, err := cmd.Flags().GetString("helm-repository-namespace")
			if err != nil {
				return err
			}
			helmRepositoryFilepath, err := cmd.Flags().GetString("helm-repository-filepath")
			if err != nil {
				return err
			}
			helmReleaseName, err := cmd.Flags().GetString("helm-release-name")
			if err != nil {
				return err
			}
			helmReleaseNamespace, err := cmd.Flags().GetString("helm-release-namespace")
			if err != nil {
				return err
			}
			helmReleaseFilepath, err := cmd.Flags().GetString("helm-release-filepath")
			if err != nil {
				return err
			}
			helmReleaseTargetNamespace, err := cmd.Flags().GetString("helm-release-target-namespace")
			if err != nil {
				return err
			}
			helmReleaseValues, err := cmd.Flags().GetStringSlice("helm-release-values")
			if err != nil {
				return err
			}
			helmReleaseValuesFrom, err := cmd.Flags().GetStringSlice("helm-release-values-from")
			if err != nil {
				return err
			}

			gitlabAgentTokenSecretName, err := cmd.Flags().GetString("gitlab-agent-token-secret-name")
			if err != nil {
				return err
			}

			fluxSourceType, err := cmd.Flags().GetString("flux-source-type")
			if err != nil {
				return err
			}

			fluxSourceNamespace, err := cmd.Flags().GetString("flux-source-namespace")
			if err != nil {
				return err
			}

			fluxSourceName, err := cmd.Flags().GetString("flux-source-name")
			if err != nil {
				return err
			}

			createEnvironment, err := cmd.Flags().GetBool("create-environment")
			if err != nil {
				return err
			}

			var environmentCfg *environmentConfiguration
			if createEnvironment {
				environmentCfg = &environmentConfiguration{
					name:                fmt.Sprintf("%s/%s", helmReleaseNamespace, helmReleaseName),
					kubernetesNamespace: helmReleaseTargetNamespace,
					fluxResourcePath:    fmt.Sprintf("helm.toolkit.fluxcd.io/v2beta1/namespaces/%s/helmreleases/%s", helmReleaseNamespace, helmReleaseName),
				}

				if cmd.Flags().Changed("environment-name") {
					environmentName, err := cmd.Flags().GetString("environment-name")
					if err != nil {
						return err
					}
					environmentCfg.name = environmentName
				}

				if cmd.Flags().Changed("environment-namespace") {
					environmentNamespace, err := cmd.Flags().GetString("environment-namespace")
					if err != nil {
						return err
					}
					environmentCfg.kubernetesNamespace = environmentNamespace
				}

				if cmd.Flags().Changed("environment-flux-resource-path") {
					environmentFluxResourcePath, err := cmd.Flags().GetString("environment-flux-resource-path")
					if err != nil {
						return err
					}
					environmentCfg.fluxResourcePath = environmentFluxResourcePath
				}
			}

			c := cf(stdout, stderr, os.Environ())

			return (&bootstrapCmd{
				api:            api,
				stdout:         stdout,
				stderr:         stderr,
				agentName:      args[0],
				manifestBranch: manifestBranch,
				kubectl:        kwf(c, kubectlBinaryName, helmReleaseTargetNamespace, gitlabAgentTokenSecretName),
				flux: fwf(
					c, fluxBinaryName, manifestPath,
					helmRepositoryName, helmRepositoryNamespace, helmRepositoryFilepath,
					helmReleaseName, helmReleaseNamespace, helmReleaseFilepath, helmReleaseTargetNamespace,
					helmReleaseValues, helmReleaseValuesFrom,
					fluxSourceType, fluxSourceNamespace, fluxSourceName,
				),
				noReconcile:    noReconcile,
				environmentCfg: environmentCfg,
			}).run()
		},
	}
	agentBootstrapCmd.Flags().StringP("manifest-path", "p", "", "Location of directory in Git repository for storing the GitLab Agent for Kubernetes Helm resources.")
	agentBootstrapCmd.Flags().StringP("manifest-branch", "b", "", "Branch to commit the Flux Manifests to. (default to the project default branch)")

	agentBootstrapCmd.Flags().Bool("no-reconcile", false, "Do not trigger Flux reconciliation for GitLab Agent for Kubernetes Flux resource.")

	agentBootstrapCmd.Flags().String("helm-repository-name", "gitlab", "Name of the Flux HelmRepository manifest.")
	agentBootstrapCmd.Flags().String("helm-repository-namespace", "flux-system", "Namespace of the Flux HelmRepository manifest.")
	agentBootstrapCmd.Flags().String("helm-repository-filepath", "gitlab-helm-repository.yaml", "Filepath within the GitLab Agent project to commit the Flux HelmRepository to.")

	agentBootstrapCmd.Flags().String("helm-release-name", "gitlab-agent", "Name of the Flux HelmRelease manifest.")
	agentBootstrapCmd.Flags().String("helm-release-namespace", "flux-system", "Namespace of the Flux HelmRelease manifest.")
	agentBootstrapCmd.Flags().String("helm-release-filepath", "gitlab-agent-helm-release.yaml", "Filepath within the GitLab Agent project to commit the Flux HelmRelease to.")
	agentBootstrapCmd.Flags().String("helm-release-target-namespace", "gitlab-agent", "Namespace of the GitLab Agent deployment.")
	agentBootstrapCmd.Flags().StringSlice("helm-release-values", nil, "Local path to values.yaml files")
	agentBootstrapCmd.Flags().StringSlice("helm-release-values-from", nil, "Kubernetes object reference that contains the values.yaml data key in the format '<kind>/<name>', where kind must be one of: (Secret,ConfigMap)")

	agentBootstrapCmd.Flags().String("gitlab-agent-token-secret-name", "gitlab-agent-token", "Name of the Secret where the token for the GitLab Agent is stored. The helm-release-target-namespace is implied for the namespace of the Secret.")

	agentBootstrapCmd.Flags().String("flux-source-type", "git", "Source type of the flux-system, e.g. git, oci, helm, ...")
	agentBootstrapCmd.Flags().String("flux-source-namespace", "flux-system", "Flux source namespace.")
	agentBootstrapCmd.Flags().String("flux-source-name", "flux-system", "Flux source name.")

	agentBootstrapCmd.Flags().Bool("create-environment", true, "Create an Environment for the GitLab Agent.")
	agentBootstrapCmd.Flags().String("environment-name", "<helm-release-namespace>/<helm-release-name>", "Name of the Environment for the GitLab Agent.")
	agentBootstrapCmd.Flags().String("environment-namespace", "<helm-release-namespace>", "Kubernetes namespace of the Environment for the GitLab Agent.")
	agentBootstrapCmd.Flags().String("environment-flux-resource-path", "helm.toolkit.fluxcd.io/v2beta1/namespaces/<helm-release-namespace>/helmreleases/<helm-release-name>", "Flux Resource Path of the Environment for the GitLab Agent.")

	return agentBootstrapCmd
}

type bootstrapCmd struct {
	api            API
	stdout         io.Writer
	stderr         io.Writer
	agentName      string
	manifestBranch string
	kubectl        KubectlWrapper
	flux           FluxWrapper
	noReconcile    bool
	environmentCfg *environmentConfiguration
}

type file struct {
	path    string
	content []byte
}

type environmentConfiguration struct {
	name                string
	kubernetesNamespace string
	fluxResourcePath    string
}

func (c *bootstrapCmd) run() error {
	// 1. Register the agent
	fmt.Fprintf(c.stderr, "Registering Agent ... ")
	agent, err := c.registerAgent()
	if err != nil {
		fmt.Fprintf(c.stderr, "[FAILED]\n")
		return err
	}
	fmt.Fprintf(c.stderr, "[OK]\n")

	// 2. Configure the Agent
	fmt.Fprintf(c.stderr, "Configuring Agent ... ")
	err = c.configureAgent(agent)
	if err != nil {
		fmt.Fprintf(c.stderr, "[FAILED]\n")
		return err
	}
	fmt.Fprintf(c.stderr, "[OK]\n")

	// 3. Configure Environment for Agent
	fmt.Fprintf(c.stderr, "Configuring Environment with Dashboard for Agent ... ")

	if c.environmentCfg != nil {
		err = c.configureEnvironment(agent)
		if err != nil {
			fmt.Fprintf(c.stderr, "[FAILED]\n")
			return err
		}
		fmt.Fprintf(c.stderr, "[OK]\n")
	} else {
		fmt.Fprintf(c.stderr, "[SKIPPED]\n")
	}

	// 4. Create a token for the registered agent
	fmt.Fprintf(c.stderr, "Creating Agent Token ... ")
	token, err := c.createAgentToken(agent)
	if err != nil {
		fmt.Fprintf(c.stderr, "[FAILED]\n")
		return err
	}
	fmt.Fprintf(c.stderr, "[OK]\n")

	// 5. Push token in Kubernetes secret to cluster
	fmt.Fprintf(c.stderr, "Creating Kubernetes Secret with Agent Token ... ")
	err = c.createAgentTokenKubernetesSecret(token)
	if err != nil {
		fmt.Fprintf(c.stderr, "[FAILED]\n")
		return err
	}
	fmt.Fprintf(c.stderr, "[OK]\n")

	// 6. Create Flux HelmRepository and HelmRelease resource.
	fmt.Fprintf(c.stderr, "Creating Flux Helm Resources ... ")
	helmResourceFiles, err := c.createFluxHelmResources()
	if err != nil {
		fmt.Fprintf(c.stderr, "[FAILED]\n")
		return err
	}
	fmt.Fprintf(c.stderr, "[OK]\n")

	// 7. Commit and Push the created Flux Helm resources to the manifest path.
	fmt.Fprintf(c.stderr, "Syncing Flux Helm Resources ... ")
	err = c.syncFluxHelmResourceFiles(helmResourceFiles)
	if err != nil {
		fmt.Fprintf(c.stderr, "[FAILED]\n")
		return err
	}
	fmt.Fprintf(c.stderr, "[OK]\n")

	// 6. Trigger Flux reconciliation of GitLab Agent HelmRelease.
	fmt.Fprintf(c.stderr, "Reconciling Flux Helm Resources ... ")
	if !c.noReconcile {
		fmt.Fprintln(c.stderr, "Output from flux command:")
		err = c.fluxReconcile()
		if err != nil {
			return reconcileErr
		}
	} else {
		fmt.Fprintf(c.stderr, "[SKIPPED]\n")
	}

	fmt.Fprintln(c.stderr, "Successfully bootstrapped the GitLab Agent")
	return nil
}

func (c *bootstrapCmd) registerAgent() (*gitlab.Agent, error) {
	agent, err := c.api.GetAgentByName(c.agentName)
	if err != nil {
		if !errors.Is(err, glab_api.AgentNotFoundErr) {
			return nil, err
		}

		// register agent
		agent, err = c.api.RegisterAgent(c.agentName)
		if err != nil {
			return nil, err
		}
	}
	return agent, nil
}

func (c *bootstrapCmd) configureAgent(agent *gitlab.Agent) error {
	return c.api.ConfigureAgent(agent, c.manifestBranch)
}

func (c *bootstrapCmd) configureEnvironment(agent *gitlab.Agent) error {
	return c.api.ConfigureEnvironment(agent.ID, c.environmentCfg.name, c.environmentCfg.kubernetesNamespace, c.environmentCfg.fluxResourcePath)
}

func (c *bootstrapCmd) createAgentToken(agent *gitlab.Agent) (*gitlab.AgentToken, error) {
	return c.api.CreateAgentToken(agent.ID)
}

func (c *bootstrapCmd) createAgentTokenKubernetesSecret(token *gitlab.AgentToken) error {
	return c.kubectl.createAgentTokenSecret(token.ID, token.Token)
}

func (c *bootstrapCmd) createFluxHelmResources() ([]file, error) {
	helmRepository, err := c.flux.createHelmRepositoryManifest()
	if err != nil {
		return nil, err
	}

	kasAddress, err := c.api.GetKASAddress()
	if err != nil {
		return nil, err
	}

	helmRelease, err := c.flux.createHelmReleaseManifest(kasAddress)
	if err != nil {
		return nil, err
	}

	return []file{helmRepository, helmRelease}, nil
}

func (c *bootstrapCmd) syncFluxHelmResourceFiles(files []file) error {
	for _, f := range files {
		err := c.api.SyncFile(f, c.manifestBranch)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *bootstrapCmd) fluxReconcile() error {
	return c.flux.reconcile()
}
