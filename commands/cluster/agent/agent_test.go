package agent

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/cli/commands/cmdtest"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/pkg/iostreams"
	"gitlab.com/gitlab-org/cli/test"
)

func TestNewCmdAgent(t *testing.T) {
	old := os.Stdout // keep backup of the real stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	assert.Nil(t, NewCmdAgent(&cmdtest.Factory{
		IOStub: &iostreams.IOStreams{
			StdOut: os.Stdout,
		},
		HttpClientStub: func() (*gitlab.Client, error) { return nil, nil },
		BaseRepoStub:   func() (glrepo.Interface, error) { return glrepo.New("OWNER", "REPO"), nil },
	}).Execute())

	out := test.ReturnBuffer(old, r, w)

	assert.Contains(t, out, "Manage GitLab Agents for Kubernetes")
}
