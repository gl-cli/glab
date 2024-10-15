package bootstrap

import (
	"bytes"
	"fmt"
)

var _ KubectlWrapper = (*localKubectlWrapper)(nil)

func NewLocalKubectlWrapper(cmd Cmd, binary string, gitlabAgentNamespace string, gitlabAgentTokenSecretName string) KubectlWrapper {
	return &localKubectlWrapper{
		cmd:                        cmd,
		binary:                     binary,
		gitlabAgentNamespace:       gitlabAgentNamespace,
		gitlabAgentTokenSecretName: gitlabAgentTokenSecretName,
	}
}

type localKubectlWrapper struct {
	cmd                        Cmd
	binary                     string
	gitlabAgentNamespace       string
	gitlabAgentTokenSecretName string
}

func (k *localKubectlWrapper) createAgentTokenSecret(tokenID int, token string) error {
	namespaceFlag := fmt.Sprintf("-n=%s", k.gitlabAgentNamespace)

	output, err := k.cmd.RunWithOutput(k.binary, "create", "namespace", k.gitlabAgentNamespace)
	if err != nil {
		if !bytes.Contains(output, []byte("already exists")) {
			return err
		}

		// let's not even bother to first check if the secret exists or not - just attempt to delete it ...
		output, err = k.cmd.RunWithOutput(k.binary, "delete", "secret", k.gitlabAgentTokenSecretName, namespaceFlag)
		if err != nil {
			if !bytes.Contains(output, []byte("not found")) {
				return err
			}
		}
	}

	// create the secret (again) with the next token
	_, err = k.cmd.RunWithOutput(
		k.binary,
		"create",
		"secret",
		"generic",
		k.gitlabAgentTokenSecretName,
		namespaceFlag,
		"--type=Opaque",
		fmt.Sprintf("--from-literal=token=%s", token),
	)
	if err != nil {
		return err
	}

	// annotate the secret with some metadata
	_, err = k.cmd.RunWithOutput(
		k.binary,
		"annotate",
		"secrets",
		k.gitlabAgentTokenSecretName,
		namespaceFlag,
		fmt.Sprintf("gitlab.com/agent-token-id=%d", tokenID),
	)
	if err != nil {
		return err
	}

	return nil
}
