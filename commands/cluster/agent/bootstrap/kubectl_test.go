package bootstrap

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestKubectl_createAgentSecretToken_NewNamespace(t *testing.T) {
	// GIVEN
	mockCmd, k := setupKubectl(t)

	gomock.InOrder(
		mockCmd.EXPECT().RunWithOutput("kubectl", "create", "namespace", "gitlab-agent"),
		mockCmd.EXPECT().RunWithOutput("kubectl", "create", "secret", "generic", "gitlab-agent-token", "-n=gitlab-agent", "--type=Opaque", "--from-literal=token=any-token"),
		mockCmd.EXPECT().RunWithOutput("kubectl", "annotate", "secrets", "gitlab-agent-token", "-n=gitlab-agent", "gitlab.com/agent-token-id=42"),
	)

	// WHEN
	err := k.createAgentTokenSecret(42, "any-token")

	// THEN
	assert.NoError(t, err)
}

func TestKubectl_createAgentSecretToken_NamespaceAlreadyExists(t *testing.T) {
	// GIVEN
	mockCmd, k := setupKubectl(t)

	gomock.InOrder(
		mockCmd.EXPECT().RunWithOutput("kubectl", "create", "namespace", "gitlab-agent").Return([]byte("already exists"), errors.New("test")),
		mockCmd.EXPECT().RunWithOutput("kubectl", "delete", "secret", "gitlab-agent-token", "-n=gitlab-agent").Return([]byte("not found"), errors.New("test")),
		mockCmd.EXPECT().RunWithOutput("kubectl", "create", "secret", "generic", "gitlab-agent-token", "-n=gitlab-agent", "--type=Opaque", "--from-literal=token=any-token"),
		mockCmd.EXPECT().RunWithOutput("kubectl", "annotate", "secrets", "gitlab-agent-token", "-n=gitlab-agent", "gitlab.com/agent-token-id=42"),
	)

	// WHEN
	err := k.createAgentTokenSecret(42, "any-token")

	// THEN
	assert.NoError(t, err)
}

func TestKubectl_createAgentSecretToken_NamespaceCreationFails(t *testing.T) {
	// GIVEN
	mockCmd, k := setupKubectl(t)

	mockCmd.EXPECT().RunWithOutput("kubectl", "create", "namespace", "gitlab-agent").Return([]byte("unknown error"), errors.New("test"))

	// WHEN
	err := k.createAgentTokenSecret(42, "any-token")

	// THEN
	assert.Error(t, err)
}

func TestKubectl_createAgentSecretToken_SecretDeletionFails(t *testing.T) {
	// GIVEN
	mockCmd, k := setupKubectl(t)

	gomock.InOrder(
		mockCmd.EXPECT().RunWithOutput("kubectl", "create", "namespace", "gitlab-agent").Return([]byte("already exists"), errors.New("test")),
		mockCmd.EXPECT().RunWithOutput("kubectl", "delete", "secret", "gitlab-agent-token", "-n=gitlab-agent").Return([]byte("unknown error"), errors.New("test")),
	)

	// WHEN
	err := k.createAgentTokenSecret(42, "any-token")

	// THEN
	assert.Error(t, err)
}

func TestKubectl_createAgentSecretToken_SecretCreationFails(t *testing.T) {
	// GIVEN
	mockCmd, k := setupKubectl(t)

	gomock.InOrder(
		mockCmd.EXPECT().RunWithOutput("kubectl", "create", "namespace", "gitlab-agent"),
		mockCmd.EXPECT().RunWithOutput("kubectl", "create", "secret", "generic", "gitlab-agent-token", "-n=gitlab-agent", "--type=Opaque", "--from-literal=token=any-token").Return([]byte("unknown error"), errors.New("test")),
	)

	// WHEN
	err := k.createAgentTokenSecret(42, "any-token")

	// THEN
	assert.Error(t, err)
}

func setupKubectl(t *testing.T) (*MockCmd, KubectlWrapper) {
	ctrl := gomock.NewController(t)
	mockCmd := NewMockCmd(ctrl)
	k := NewLocalKubectlWrapper(mockCmd, "kubectl", "gitlab-agent", "gitlab-agent-token")

	return mockCmd, k
}
